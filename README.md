Incident Reviewer
=================

![Mockup of the interface](./docs/images/incident-reviewer.excalidraw.svg#gh-light-mode-only)
![Mockup of the interface](./docs/images/incident-reviewer.dark.excalidraw.svg#gh-dark-mode-only)

## Development

For all the scripts in `script/` you can see what they do at a high level by running:

```shell
grep '##' script/*
```

To install dependencies needed run:

```shell
script/bootstrap
```

If you pull and things don't work as expected, to update run:

```shell
script/update
```

Start the local dev server by running:

```shell
script/server
```

### Using with Colima

If you are using Colima instead of Docker for running your pods you need to add some config to make testcontainers work.
[The solution](https://github.com/testcontainers/testcontainers-java/issues/6450#issuecomment-1814342263) is to:

1. `sudo ln -s /Users/$USER/.colima/docker.sock /var/run/docker.sock`, to make sure the colima socket is available where tools expect it by default
2. `colima stop && colima start --network-address`, to make sure Colima is exposing an IP address to your computer
3. Get the IP address colima is exposing: `colima ls` copy from the address field
4. Make these environment variables available, for example in `.bash_profile`:
   ```
   export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
   export TESTCONTAINERS_HOST_OVERRIDE=<paste IP address from colima ls>
   ```
5. Run `script/test` and it should all work
