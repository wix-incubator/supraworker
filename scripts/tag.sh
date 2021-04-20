#!/usr/bin/env bash
version=${1:-}
if [[ -z "$version" ]]; then
    echo "Please use: $0 1.0.0-alpha1"
    exit 1
fi

function check_version() {
  if [[ $1 =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$ ]]; then
    echo "$1"
  else
    echo ""
  fi
}

function check_version_ex() {
    if [[ $1 =~ ^vv.+$ ]]; then
        check_version "${1:2}"
    elif [[ $1 =~ ^v.+$ ]]; then
        check_version "${1:1}"
    else
        check_version "${1}"
  fi
}

semver=$(check_version_ex "$version")


echo "Check a version [${version}] on compliance to Semantic Versioning 2.0.0. (https://semver.org/)"

if [[ ! "$semver" ]]; then
    echo "'$version' is not a valid semantic version"
    exit 2
fi
git show-ref --tags -d|grep -q "refs/tags/v${version}"'$'
if [[ $? -eq 0 ]];then
    echo "Tag v${version} is already exist"
    exit 1
fi

git tag -d v${version}
git push origin :refs/tags/v${version}
git tag -a v${version} -m "version ${version}" \
&& git push origin v${version}
exit_code=$?
exit ${exit_code:-127}