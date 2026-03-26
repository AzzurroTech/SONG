document.addEventListener('DOMContentLoaded', () => {
    const form = document.getElementById('dataForm');
    const responseDiv = document.getElementById('response');

    if (form) {
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(form);
            
            try {
                const res = await fetch('/api/submit', {
                    method: 'POST',
                    body: formData
                });
                const data = await res.json();
                responseDiv.innerHTML = `<p style="color: green;">${data.message}</p>`;
            } catch (err) {
                responseDiv.innerHTML = `<p style="color: red;">Error submitting data.</p>`;
            }
        });
    }
});