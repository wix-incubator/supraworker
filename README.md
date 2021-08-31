# <img src="./images/logo-white.png" data-canonical-src="./images/logo-white.png" width="80px" height="80px" style="background-color: black;" /> Supraworker - Pull Jobs from Anywhere
![GitHub All Releases](https://img.shields.io/github/downloads/wix/supraworker/total) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Go Report Card](https://goreportcard.com/badge/github.com/wix/supraworker)](https://goreportcard.com/report/github.com/wix/supraworker)


Supraworker is an abstraction layer around jobs, allowing one to pull any job from any API, callback your APIs, observe execution times and control concurrent execution.

> TL;DR: Supraworker makes an API request for a job => API responds with a bash command => Supraworker executes a bash command. Easy.

It can pull any commands (jobs) from your APIs, run commands, and stream logs back to your API. It can also send state updates to a remote API.

## Getting started

Prerequisites:
1. An API service that can serve jobs:
    * You can check this simple implantation of an [API service written in Python](docker-image/apiserver/app/app.py)
2. Supraworker configuration:
    * You can check [the configuration](tests/supraworker/supraworker.yml) for the [above API service written in Python](docker-image/apiserver/app/app.py)
3. A server to run Supraworker on  

## Installation
### MacOs X

#### Binary install
Homebrew is a free and open-source package management system for Mac OS X.
```bash
 brew tap wix/tap
```
if you are using ssh then
```shell
brew tap wix/tap  git@github.com:wix/homebrew-brew.git
```

```shell
brew update
brew install wix/tap/supraworker
```


To update to the latest, run
```bash
brew upgrade wix/tap/supraworker
```

#### Installing from source
Find the version you wish to install on the [GitHub Releases
page](https://github.com/wix/supraworker/releases) and download either the
`darwin-amd64` binary for MacOS or the `linux-amd64` binary for Linux. No pre-built binaries are provided for other operating systems or architectures at this time.

> **Running releases on MacOS:**

> You need to the download file, extract it, and remove attributes with the following command (where ~/Downloads/supraworker_darwin_amd64/supraworker is path to the file):

```bash
$ xattr -d com.apple.quarantine ~/Downloads/supraworker_darwin_amd64/supraworker
$ ~/Downloads/supraworker_darwin_amd64/supraworker
```

### Linux
#### Download the latest release
```bash
curl --silent -L  "https://api.github.com/repos/wix/supraworker/releases/latest"  \
| jq --arg PLATFORM_ARCH "$(echo `uname -s`_amd| tr '[:upper:]' '[:lower:]')" -r '.assets[] | select(.name | contains($PLATFORM_ARCH)).browser_download_url' \
| xargs -I % curl -sSL  % \
| sudo tar --strip-components=1  -xzf  -
```

#### Installing from source

1. install [Go](http://golang.org) `v1.13+`
1. clone into your `$GOPATH`:

    ```bash
    mkdir -p $GOPATH/src/github.com/wix
    git clone https://github.com/wix/supraworker $GOPATH/src/github.com/wix/supraworker
    cd $GOPATH/src/github.com/wix/supraworker
    ```
1. install [golangci-lint](https://github.com/golangci/golangci-lint#install) for linting + static analysis
   * Lint: `docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.24.0 golangci-lint run -v`


### Configuration
Define the config file at `$HOME/supraworker.yaml`:

---
> **_NOTE:_** Keys are not case-sensitive (https://github.com/spf13/viper/pull/635#issuecomment-580659676):
Viper's default behaviour of lowercasing keys for key insensitivity is incompatible with these standards (like when using case-sensitive API credentials).
For eg: MyApiKey=MySecret
---

```yaml
# ClientId in case you need to identify the worker
clientId: "my_uniq_client_id"
# how often to call your API, in seconds
api_delay_sec: 10
# operations related to the jobs
jobs:
  get:
    url: "http://localhost:80/get/new/job"
    method: POST
    headers:
      "Content-type": "application/json"
      params:
        "clientid": "{{ .ClientId}}"
```
## Use Cases
* Airflow sends a task to be executed on AWS EMR
* You're building your CI/CD system

## Running tests

*  expires all test results

```bash
$ go clean -testcache
```
* runs all tests

```bash
$ go test -bench= -test.v  ./...
```