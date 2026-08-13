# syncline-collection-tracking

## Run with Docker Desktop

Build the image from the repository root:

```powershell
docker build -t syncline-collection-tracking:local .
```

Copy `docker.env.example` to an untracked file named `docker.env` and set the SQL Server credentials. The default `HOST` value assumes SQL Server is running on the Docker Desktop host. If SQL Server runs in another container, use that container's network hostname instead.

Apply database migrations:

```powershell
docker run --rm `
  --name syncline-collection-tracking-migrate `
  --env-file docker.env `
  --entrypoint /migrate `
  syncline-collection-tracking:local
```

Start the application in detached mode:

```powershell
docker run -d `
  --name syncline-collection-tracking `
  --restart unless-stopped `
  --publish 8080:8080 `
  --env-file docker.env `
  syncline-collection-tracking:local
```

Open `http://localhost:8080`. The health endpoint is available at `http://localhost:8080/health`.

Stop and remove the application container when needed:

```powershell
docker rm --force syncline-collection-tracking
```
