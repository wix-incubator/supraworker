#!/usr/bin/env bash
cd "$(dirname "$0")" || true
on_exit(){
  local exitcode=$?
  docker-compose down -v
  exit ${exitcode:-1}
}
trap 'on_exit $?' SIGINT EXIT INT

docker-compose up -d

export PYTHONPATH=$PTHONPATH:./It/
python3 -m unittest discover -p 'test_*.py'
ret=$?
docker-compose down
exit $ret