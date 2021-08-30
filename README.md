# Supraworker - Pull Jobs from Anywhere
![GitHub All Releases](https://img.shields.io/github/downloads/wix/supraworker/total) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Go Report Card](https://goreportcard.com/badge/github.com/wix/supraworker)](https://goreportcard.com/report/github.com/wix/supraworker)
<img src="./images/logo.png" data-canonical-src="./images/logo.png" width="38" height="38" />

The abstraction layer around jobs, allows pull a job from any API, call-back your API, observe execution time and to control concurrent execution.

It's responsible for pulling the commands(jobs) from your API, running commands, and streaming the logs back to your API. 
It also sends state updates to remote API.

## Getting started

Prerequisite:
1. API service that serve jobs:
    * You can check the simple implantation of the [API service written ib Python](docker-image/apiserver/app/app.py)
2. Supraworker configuration:
    * You can check [the configuration](tests/supraworker/supraworker.yml) for the [above API service written ib Python](docker-image/apiserver/app/app.py)  
3. Running supraworker on a server 

## Installation 
### MacOs X

#### Binary installation 
Homebrew is a free and open-source package management system for Mac OS X.
```bash
 brew tap wix/tap
```
if you are using ssh than
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

#### Installation from source code

* Find the version you wish to install on the [GitHub Releases
page](https://github.com/wix/supraworker/releases) and download either the
`darwin-amd64` binary for MacOS or the `linux-amd64` binary for Linux. No other
operating systems or architectures have pre-built binaries at this time.

> **_NOTE:_** Running releasses on MacOs:
> You need to download file, extract it and remove attributes with
> the following command (where ~/Downloads/supraworker_darwin_amd64/supraworker is Path to the file)

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
1. clone this down into your `$GOPATH`
* `mkdir -p $GOPATH/src/github.com/wix`
* `git clone https://github.com/wix/supraworker $GOPATH/src/github.com/wix/supraworker`
* `cd $GOPATH/src/github.com/wix/supraworker`
1. install [golangci-lint](https://github.com/golangci/golangci-lint#install) for linting + static analysis
* Lint: `docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.24.0 golangci-lint run -v`


### Configuration
Define config at `$HOME/supraworker.yaml`:

---
> **_NOTE:_** Keys are not case sensitivity (https://github.com/spf13/viper/pull/635#issuecomment-580659676):
Viper's default behaviour of lowercasing the keys for key insensitivity is incompatible
with these standards and has the side effect of making it difficult for
use cases such as case sensitive API credentials in configuration.
For eg: MyApiKey=MySecret
---

```yaml
# ClientId in case you need to identify the worker
clientId: "my_uniq_client_id"
# how ofen call your API
api_delay_sec: 10
# jobs related operations
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
* Airflow sends task for an execution on AWS EMR
* Building your CI/CD system

## Running tests

*  expires all test results

```bash
$ go clean -testcache
```
* run all tests

```bash
$ go test -bench= -test.v  ./...
```
