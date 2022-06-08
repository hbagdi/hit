# hit

![image](https://hit.yolo42.com/android-chrome-512x512.png)

[![Build](https://github.com/hbagdi/hit/actions/workflows/test.yaml/badge.svg?branch=main&event=push)](https://github.com/hbagdi/hit/actions/workflows/test.yaml)
<a href="https://twitter.com/intent/follow?screen_name=hitcmd">
<img src="https://img.shields.io/twitter/follow/hitcmd?style=social&logo=twitter" alt="follow on Twitter">
</a>

Make and manage HTTP requests

Hit is a command-line program that makes it easy to build HTTP requests 
using plain-text files and execute them.

Some features include:
- authoring HTTP requests using plain-text files
- using output of one HTTP request as input to the next request
- define request templates that can be dynamically populated
- execute complex workflows by changing the execution order of requests and
  using inbuilt cache to inject a subset of response into a request
- Combine responses from multiple requests and send them in a single request

[Status](#status) | [Install](#install) | [Documentation](#documentation) |
[Getting help](#getting-help) | [Contributing](#contributing) | [License](#license)

## Status

`hit` is in early development. Expect rough edges and all feedback is welcome.

## Install

`hit` is a single statically compiled binary. The binaries are hosted on GitHub.

```shell
# macOS
brew install hbagdi/tap/hit

# Linux
curl -sL https://github.com/hbagdi/hit/releases/download/v0.2.0/hit_0.2.0_linux_amd64.tar.gz \
  -o /tmp/hit.tar.gz
tar -xf /tmp/hit.tar.gz -C /tmp
sudo cp /tmp/hit /usr/local/bin/
```

## Get started

Create a hit file:
```shell
echo '@_global
base_url=https://nodes.yolo42.com
version=1


@gen-root-node
POST
/v1/node
~y2j
title: my-root-node
' > quick-start.hit
```

Execute your first request:
```shell
hit @gen-root-node
```

For a further complete demo, please go through the
[quick-start guide](https://hit.yolo42.com/docs/get-started/quick-start/).

## Documentation 

Documentation is available at [hit.yolo42.com](https://hit.yolo42.com).

## Getting help

If you need help, please open a [GitHub issue](https://github.com/hbagdi/hit/issues/new).

## Contributing

`hit` is in early development and hence the code is evolving at a rapid pace and
there is little to none developer documentation for the code/architecture.
Patches are welcome. If you would like to make a larger change, please open 
a GitHub issue first to discuss your proposal.

## License

`hit` is licensed under [Apache 2.0](https://github.com/hbagdi/hit/blob/main/LICENSE).