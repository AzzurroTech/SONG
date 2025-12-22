package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strings"
)

// ---------- Data structures ----------
type Container struct {
	ID      string
	Image   string
	Command string
	Created string
	Status  string
	Ports   string
	Names   string
}

// Minimal subset of `docker inspect` output we need.
type inspectInfo struct {
	Config struct {
		Image string   `json:"Image"`
		Env   []string `json:"Env"`
	} `json:"Config"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

// Docker‑Hub tag list response (only a few fields we need).
type TagResult struct {
	Count   int `json:"count"`
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

// ---------- Helper functions ----------
func runDockerPS() ([]Container, error) {
	cmd := exec.Command(
		"docker", "ps",
		"--no-trunc",
		`--format`,
		`{{.ID}}\t{{.Image}}\t{{.Command}}\t{{.CreatedAt}}\t{{.Status}}\t{{.Ports}}\t{{.Names}}`,
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run docker ps: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	containers := make([]Container, 0, len(lines))
	for _, line := range lines {
		f := strings.SplitN(line, "\t", 7)
		if len(f) < 7 {
			continue // ignore malformed lines
		}
		containers = append(containers, Container{
			ID:      f[0],
			Image:   f[1],
			Command: f[2],
			Created: f[3],
			Status:  f[4],
			Ports:   f[5],
			Names:   f[6],
		})
	}
	return containers, nil
}

// Pull up to 10 tags from Docker Hub for a given image name.
func fetchTags(image string) ([]string, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/library/%s/tags?page_size=10", image)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub returned %d", resp.StatusCode)
	}
	var tr TagResult
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	tags := make([]string, 0, tr.Count)
	for _, r := range tr.Results {
		tags = append(tags, r.Name)
	}
	return tags, nil
}

// Build a minimal docker‑compose.yml from the currently running containers.
func buildComposeYAML() (string, error) {
	// 1️⃣ Get IDs of all running containers.
	cmdIDs := exec.Command("docker", "ps", "-q")
	var idsOut bytes.Buffer
	cmdIDs.Stdout = &idsOut
	if err := cmdIDs.Run(); err != nil {
		return "", fmt.Errorf("cannot list containers: %w", err)
	}
	ids := strings.Fields(idsOut.String())
	if len(ids) == 0 {
		return "# No containers are running – nothing to export.\n", nil
	}

	services := make([]string, 0, len(ids))

	for _, id := range ids {
		inspectCmd := exec.Command("docker", "inspect", id)
		var inspectOut bytes.Buffer
		inspectCmd.Stdout = &inspectOut
		if err := inspectCmd.Run(); err != nil {
			return "", fmt.Errorf("inspect failed for %s: %w", id, err)
		}
		var infoArr []inspectInfo
		if err := json.Unmarshal(inspectOut.Bytes(), &infoArr); err != nil {
			return "", fmt.Errorf("json decode failed for %s: %w", id, err)
		}
		if len(infoArr) == 0 {
			continue
		}
		info := infoArr[0]

		// Service name – deterministic identifier.
		serviceName := fmt.Sprintf("svc_%s", id[:12])

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("  %s:\n", serviceName))
		sb.WriteString(fmt.Sprintf("    image: %s\n", info.Config.Image))

		// Ports ---------------------------------------------------------
		if len(info.NetworkSettings.Ports) > 0 {
			portLines := []string{}
			for containerPortProto, bindings := range info.NetworkSettings.Ports {
				parts := strings.Split(containerPortProto, "/")
				containerPort := parts[0]
				if len(bindings) == 0 {
					// No host binding – expose the same port.
					portLines = append(portLines, fmt.Sprintf("%s:%s", containerPort, containerPort))
				} else {
					for _, b := range bindings {
						hostPort := b.HostPort
						portLines = append(portLines, fmt.Sprintf("%s:%s", hostPort, containerPort))
					}
				}
			}
			if len(portLines) > 0 {
				sort.Strings(portLines)
				sb.WriteString("    ports:\n")
				for _, pl := range portLines {
					sb.WriteString(fmt.Sprintf("      - \"%s\"\n", pl))
				}
			}
		}

		// Environment variables -------------------------------------------
		if len(info.Config.Env) > 0 {
			sb.WriteString("    environment:\n")
			for _, ev := range info.Config.Env {
				sb.WriteString(fmt.Sprintf("      - %s\n", ev))
			}
		}
		services = append(services, sb.String())
	}

	var final strings.Builder
	final.WriteString("version: \"3\"\n")
	final.WriteString("services:\n")
	for _, s := range services {
		final.WriteString(s)
	}
	return final.String(), nil
}

// ---------- HTTP Handlers ----------
func indexHandler(w http.ResponseWriter, r *http.Request) {
	containers, err := runDockerPS()
	if err != nil {
		http.Error(w, "Cannot run `docker ps`. Ensure Docker is installed and you have permission.", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, map[string]interface{}{
		"Containers": containers,
	})
}

// POST /run – start a container from the supplied image.
func runContainerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	image := strings.TrimSpace(r.FormValue("image"))
	if image == "" {
		http.Error(w, "Image name required", http.StatusBadRequest)
		return
	}
	if strings.ContainsAny(image, " \t\r\n\"'`$&|<>") {
		http.Error(w, "Invalid image name", http.StatusBadRequest)
		return
	}
	cmd := exec.Command("docker", "run", "-d", image)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		msg := fmt.Sprintf("Failed to start container: %s – %s", err, out.String())
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// POST /stop – stop a single container.
func stopContainerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		http.Error(w, "Container ID required", http.StatusBadRequest)
		return
	}
	cmd := exec.Command("docker", "stop", id)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		msg := fmt.Sprintf("Failed to stop container %s: %s – %s", id, err, out.String())
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// POST /stopall – stop **all** running containers.
func stopAllHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cmdIDs := exec.Command("docker", "ps", "-q")
	var idsOut bytes.Buffer
	cmdIDs.Stdout = &idsOut
	if err := cmdIDs.Run(); err != nil {
		http.Error(w, "Failed to list containers", http.StatusInternalServerError)
		return
	}
	ids := strings.Fields(idsOut.String())
	if len(ids) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	var failed []string
	for _, id := range ids {
		stopCmd := exec.Command("docker", "stop", id)
		var out bytes.Buffer
		stopCmd.Stdout = &out
		stopCmd.Stderr = &out
		if err := stopCmd.Run(); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%s)", id, strings.TrimSpace(out.String())))
		}
	}
	if len(failed) > 0 {
		msg := fmt.Sprintf("Some containers could not be stopped: %s", strings.Join(failed, ", "))
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// GET /tags?image=nginx – return a JSON array of tag strings.
func tagsAPIHandler(w http.ResponseWriter, r *http.Request) {
	image := strings.TrimSpace(r.URL.Query().Get("image"))
	if image == "" {
		http.Error(w, "image query param required", http.StatusBadRequest)
		return
	}
	tags, err := fetchTags(image)
	if err != nil {
		http.Error(w, "Could not fetch tags from Docker Hub", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// GET /export – download a docker‑compose.yml for the current containers.
func exportComposeHandler(w http.ResponseWriter, r *http.Request) {
	yaml, err := buildComposeYAML()
	if err != nil {
		http.Error(w, "Failed to build compose file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", `attachment; filename="docker-compose.yml"`)
	io.WriteString(w, yaml)
}

// ---------- HTML Template ----------
var tmpl = template.Must(template.New("page").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>SONG – Docker Demo Server</title>
<style>
	body {font-family:Arial,sans-serif;background:#fafafa;margin:20px;}
	h1 {color:#222;}
	table {border-collapse:collapse;width:100%;max-width:1200px;}
	th, td {padding:8px 12px;border:1px solid #ddd;text-align:left;}
	th {background:#4a90e2;color:#fff;}
	tr:nth-child(even){background:#f2f2f2;}
	.plus-btn, .minus-btn, .export-btn {
		font-size:24px;cursor:pointer;border:none;border-radius:50%;
		width:48px;height:48px;display:flex;align-items:center;justify-content:center;
		position:fixed;bottom:30px;box-shadow:0 2px 6px rgba(0,0,0,.2);
	}
	.plus-btn {background:#28a745;color:#fff;right:30px;}
	.minus-btn {background:#dc3545;color:#fff;right:90px;}
	.export-btn {background:#0069d9;color:#fff;right:150px;}
	.overlay {position:fixed;top:0;left:0;width:100%;height:100%;
		background:rgba(0,0,0,.5);display:none;align-items:center;justify-content:center;}
	.modal {background:#fff;padding:20px;border-radius:6px;box-shadow:0 2px 8px rgba(0,0,0,.3);
		min-width:300px;}
	.modal h2 {margin-top:0;}
	.modal input[type=text] {width:100%;padding:6px;margin:6px 0;}
	.modal button {margin-top:8px;padding:6px 12px;}
	.tag-list {margin-top:8px;}
	.tag-item {display:inline-block;background:#e9ecef;padding:4px 8px;margin:2px;
		border-radius:4px;cursor:pointer;}
</style>
</head>
<body>
<h1>Running Docker Containers</h1>

{{if .Containers}}
<table>
	<tr>
		<th></th><!-- per‑row stop button -->
		<th>ID</th><th>Image</th><th>Command</th><th>Created</th>
		<th>Status</th><th>Ports</th><th>Name(s)</th>
	</tr>
	{{range .Containers}}
	<tr>
		<td>
			<form method="POST" action="/stop" style="margin:0;">
				<input type="hidden" name="id" value="{{.ID}}">
				<button type="submit" title="Stop container {{.ID}}" style="background:none;border:none;color:#dc3545;font-weight:bold;cursor:pointer;">&#x2212;</button>
			</form>
		</td>
		<td style="font-family:monospace;">{{.ID}}</td>
		<td>{{.Image}}</td>
		<td>{{.Command}}</td>
		<td>{{.Created}}</td>
		<td>{{.Status}}</td>
		<td>{{.Ports}}</td>
		<td>{{.Names}}</td>
	</tr>
	{{end}}
</table>
{{else}}
<p>No containers are currently running.</p>
{{end}}

<!-- + button – add new container -->
<button class="plus-btn" id="openModal">+</button>

<!-- large red button – stop ALL containers -->
<form method="POST" action="/stopall" style="display:inline;">
	<button class="minus-btn" type="submit" title="Stop all running containers">&#x2212;</button>
</form>

<!-- export compose button -->
<a href="/export" class="export-btn" title="Export current setup as docker‑compose.yml">⤓</a>

<!-- Modal overlay – add container -->
<div class="overlay" id="modalOverlay">
	<div class="modal">
		<h2>Run a new container</h2>
		<label for="imageInput">Docker‑Hub image (e.g. <code>nginx</code> or <code>redis:alpine</code>)</label>
		<input type="text" id="imageInput" placeholder="image name">

		<button id="searchBtn">Search tags</button>

		<div class="tag-list" id="tagList"></div>

		<form method="POST" action="/run" id="runForm" style="margin-top:12px;">
			<input type="hidden" name="image" id="selectedImage">
			<button type="submit">Run</button>
			<button type="button" id="closeBtn">Cancel</button>
		</form>
	</div>
</div>

<script>
// Show/hide the “add container” modal
document.getElementById('openModal').onclick = () => {
	document.getElementById('modalOverlay').style.display = 'flex';
};
document.getElementById('closeBtn').onclick = () => {
	document.getElementById('modalOverlay').style.display = 'none';
};

// Tag search – contacts /tags endpoint
document.getElementById('searchBtn').onclick = async () => {
	const img = document.getElementById('imageInput').value.trim();
	if (!img) return alert('Enter an image name first');
	const resp = await fetch('/tags?image=' + encodeURIComponent(img));
	if (!resp.ok) return alert('Could not fetch tags');
	const tags = await resp.json();
	const list = document.getElementById('tagList');
	list.innerHTML = '';
	tags.forEach(t => {
		const span = document.createElement('span');
		span.textContent = t;
		span.className = 'tag-item';
		span.onclick = () => {
			document.getElementById('selectedImage').value = img + ':' + t;
			document.getElementById('runForm').requestSubmit();
		};
		list.appendChild(span);
	});
};
</script>
</body>
</html>
`))

// ---------- main ----------
func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/run", runContainerHandler)
	http.HandleFunc("/stop", stopContainerHandler)
	http.HandleFunc("/stopall", stopAllHandler)
	http.HandleFunc("/tags", tagsAPIHandler)
	http.HandleFunc("/export", exportComposeHandler)

	fmt.Println("Server listening on http://localhost:8042")
	if err := http.ListenAndServe(":8042", nil); err != nil {
		panic(err)
	}
}