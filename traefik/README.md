# Deployment with traefik

To deploy the CalmNews service behind Traefik with HTTPS and Basic Authentication, follow these steps:

### 1. Overview

This deployment uses Traefik as a reverse proxy and automatic SSL manager, along with Docker Compose. Both CalmNews and Traefik containers will run on the same Docker network. Traefik securely exposes CalmNews at `https://calmnews.iotitlan.com` and the Traefik dashboard at `https://traefik.iotitlan.com`. 

Sensitive endpoints (the settings page and the Traefik dashboard) are protected with HTTP Basic Authentication.

---

### 2. Prerequisites

- **DNS:** Point both `calmnews.iotitlan.com` and `traefik.iotitlan.com` to your server's IP.
- **Docker and Docker Compose:** Follow the install instructions below for a typical Ubuntu setup.
- **A public email address:** For Let's Encrypt certificate issuance.

---

### 3. Credentials (Basic Auth)

Before you start, generate a password hash for your admin user:

```bash
htpasswd -nbm admin yourpassword
```

The output looks like: `admin:$2y$05$abcd...etc`.  
You will use this value to set credentials in config files.

---

### 4. Directory Structure Example

```
/opt/calmnews/
  ├── data/               # Persistent data for CalmNews
  ├── traefik/
        ├── dynamic_conf.yml
        ├── docker-compose.yml
```

---

### 5. Configuration Steps

#### a. `docker-compose.yml`

- The compose file defines two services: `calmnews` and `traefik`.
- Replace `<CREDENTIALS>` in both labels and `dynamic_conf.yml` with the string you generated above.

#### b. `dynamic_conf.yml`

- Configures Traefik dashboard access and HTTPS redirection rules.
- Make sure to set `<CREDENTIALS>` accordingly.

---

### 6. Deploy the Stack

Navigate to your traefik directory:
```bash
cd /opt/calmnews/traefik
docker compose up -d
```

Traefik will obtain and auto-renew HTTPS certificates via Let's Encrypt on first access.

---

### 7. Accessing Services

- **CalmNews app:** https://calmnews.iotitlan.com  
  The main app is public, but the `/settings` path requires login (protected via Basic Auth).
- **CalmNews settings:** https://calmnews.iotitlan.com/settings  
  You will be prompted for the admin username and password you set up.
- **Traefik Dashboard:** https://traefik.iotitlan.com  
  Also protected with Basic Auth.

---

### 8. Security Notes

- Only HTTPS (port 443) is exposed via Traefik to the public.
- HTTP requests (port 80) are automatically redirected to HTTPS.
- No ports other than 443 (HTTPS) and, optionally, 80 (HTTP for redirect/Let's Encrypt), should be accessible to the outside.
- All persistent data for CalmNews lives in `/opt/calmnews/data`.

---

### 9. Customization

- **Change domain name:** Edit all `.yml` files accordingly.
- **Add more users:** Add additional `htpasswd` lines, separated by commas, to the credentials field.
- **Email for certificates:** Edit `--certificatesresolvers.letsencrypt.acme.email` in `docker-compose.yml`.

For further customization, consult the sample `docker-compose.yml` and `dynamic_conf.yml` in this directory.

---



