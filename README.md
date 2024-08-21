# POD
PROTON-OTHER-DOCKER Stack

##Droplet Data
RAM: 2GB+
CPU: 2+
Backup: TRUE

##How to (DigitalOcean is assumed for this document)
1) Create a DO Droplet using the Docker template
2) SSH into the Droplet
3) install git
4) RUN "git clone https://github.com/AzzurroTech/POD.git"
5) CD into POD
6) Review the CaddyFile for the correct domain, edit as needed 
7) Create a link between hosts /etc/caddy and /POD/etc/caddy/Caddyfile using this command "ln -s /POD/etc/caddy/Caddyfile /etc/caddy"
8) Run "docker compose up -d"
9) Follow Odoo setup instructions

NOTE: SHOULD HTTPS NOT WORK
Run the following commands replacing w,x,y,z with your instance IP
docker exec pod-caddy-1 caddy reverse-proxy --from example.com --to 0.0.0.0:8069
docker exec pod-caddy-1 caddy reverse-proxy --from example.com --to w.x.y.z:8069
