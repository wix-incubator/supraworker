"""
Logs management.

All functions for read, list, delete and append logs.
"""

import traceback
import os
import logging
import urllib.parse

from flask import jsonify
from flask import Blueprint, current_app, make_response
from flask_restx import Api, Resource, fields, reqparse

logger = logging.getLogger(__name__)
logs_page = Blueprint('run_log_page', __name__)
log_api = Api(
    logs_page,
    version='1.0',
    title='Logs API',
    default='logs',
    default_label='Jobs log related operations',
    description='The Jobs API allows you to operate with jobs logs. ',
    url_prefix='/job_log_api'
)
model_validation_response = log_api.model('ValidationModel', {
    'message': fields.String(required=False, description='Error message'),
})


def _params_to_filename(job_id, run_id, extra_run_id):
    """
    Prepare file name based on params.

    Args:
        job_id (str): Job Uid
        run_id (str): Job Run Uid
        extra_run_id (str): Extra Job Uid

    Returns:
        str: filenames
    """

    with current_app.app_context():
        _logs_store_dir = current_app.config.get('LOG_STORE_FOLDER', f"/tmp/")
        try:
            if not os.path.isdir(_logs_store_dir):
                os.makedirs(_logs_store_dir, exist_ok=True)
        except OSError as error:
            logger.info('Directory {} can not be created due {}'.format(
                _logs_store_dir, error))

        jobs_logs_path = f"{_logs_store_dir}/{job_id}_{run_id}_{extra_run_id}.log"

        return jobs_logs_path


def _params_to_filenames(job_id, run_id, extra_run_id):
    """
    Prepare file names based on params.

    Args:
        job_id (str): Job Uid
        run_id (str): Job Run Uid
        extra_run_id (str): Extra Job Uid

    Returns:
        list: file names
    """

    with current_app.app_context():
        _logs_store_dir = current_app.config.get('LOG_STORE_FOLDER', f"/tmp/")
        ret = [f"{_logs_store_dir}/{job_id}_{run_id}_{extra_run_id}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{run_id}_{extra_run_id}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{run_id}_{urllib.parse.quote(extra_run_id)}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{urllib.parse.quote(run_id)}_{extra_run_id}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{urllib.parse.quote(run_id)}_{urllib.parse.quote(extra_run_id)}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{urllib.parse.quote(run_id)}_{extra_run_id}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{urllib.parse.quote(run_id)}_{urllib.parse.quote(extra_run_id)}.log",
               f"{_logs_store_dir}/{urllib.parse.quote(job_id)}_{run_id}_{urllib.parse.quote(extra_run_id)}.log",
               f"{_logs_store_dir}/{job_id}_{run_id}_{urllib.parse.quote(extra_run_id)}.log",
               f"{_logs_store_dir}/{job_id}_{run_id}_{urllib.parse.unquote(extra_run_id)}.log"]

        return ret


@log_api.route('/log/stream', endpoint='run_stream_log')
class LogStreams(Resource):
    """Job Run Logs"""

    @log_api.produces('text/plain')
    @log_api.doc(params={
        'job_id': 'The canonical Job identifier for the request',
        'run_id': 'The canonical Run identifier for the request',
        'extra_run_id': 'The canonical Extra Run identifier for the request',
        'offset': 'num lines already received from the API',
        'limit': 'Maximum lines per response'
    })
    @log_api.response(400, 'Post data Validation Error', model_validation_response)
    def get(self):
        """Get Job Run Logs"""
        parser = reqparse.RequestParser()
        parser.add_argument('job_id', required=True, type=str, default='',
                            help="The Job canonical identifier for the request")
        parser.add_argument('run_id', required=True, type=str, default='',
                            help="The Run canonical identifier for the request")
        parser.add_argument('extra_run_id', required=True, type=str, default='',
                            help="The extra identifier for the run")
        parser.add_argument('offset', required=False, type=int, default=0,
                            help="num lines already received from the API")
        parser.add_argument('limit', required=False, type=int, default=100, help="Maximum lines per response")
        args = parser.parse_args()
        fn = _params_to_filename(
            job_id=args.job_id,
            run_id=args.run_id,
            extra_run_id=args.extra_run_id
        )
        ret = []
        status_code = 200
        try:
            for fn in _params_to_filenames(job_id=args.job_id,
                                           run_id=args.run_id,
                                           extra_run_id=args.extra_run_id
                                           ):

                if os.path.exists(fn):
                    with open(fn, "r") as fp:
                        status_code = 200
                        for cnt, line in enumerate(fp):
                            if cnt >= args['offset']:
                                try:
                                    ret.append(line.decode("utf-8", errors="ignore"))
                                except AttributeError:
                                    ret.append(line)
                            if len(ret) >= args['limit']:
                                break

                    return jsonify(ret)
            _files = _params_to_filenames(job_id=args.job_id,
                                          run_id=args.run_id,
                                          extra_run_id=args.extra_run_id
                                          )
            logger.info(f"Can't find the following files {_files}\n")
        except Exception as e:
            ret = {}
            status_code = 500
            error_type = type(e).__name__
            logger.debug(f"Got error {error_type} {e} {traceback.print_stack(limit=10)}")
            ret['error_msg'] = f"{error_type}: {e}"
            ret['has_error'] = True

        return ret, status_code


@log_api.route('/run', endpoint='run_log')
class LogUploader(Resource):
    """Job Run Logs"""

    @log_api.produces('text/plain')
    @log_api.doc(params={
        'job_id': 'The canonical Job identifier for the request',
        'run_id': 'The canonical Run identifier for the request',
        'extra_run_id': 'The canonical Extra Run identifier for the request'
    })
    @log_api.response(400, 'Post data Validation Error', model_validation_response)
    def get(self):
        """Get Job Run Logs"""
        parser = reqparse.RequestParser()
        parser.add_argument('job_id', required=True, type=str, default='',
                            help="The Job canonical identifier for the request")
        parser.add_argument('run_id', required=True, type=str, default='',
                            help="The Run canonical identifier for the request")
        parser.add_argument('extra_run_id', required=True, type=str, default='',
                            help="The extra identifier for the run")
        args = parser.parse_args()
        fn = _params_to_filename(
            job_id=args.job_id,
            run_id=args.run_id,
            extra_run_id=args.extra_run_id
        )
        ret = {
            "error_msg": "",
            "has_error": False

        }
        status_code = 404
        try:
            if os.path.exists(fn):
                with open(fn, "r") as myfile:
                    response = make_response(myfile.read())
                    response.headers['Content-Type'] = 'text/plain'
                    return response
        except Exception as e:
            error_type = type(e).__name__
            logger.debug(f"Got error {error_type} {e} {traceback.print_stack(limit=10)}")
            ret['error_msg'] = f"{error_type}: {e}"
            ret['has_error'] = True

        return ret, status_code

    @log_api.doc(params={
        'job_id': 'The canonical Job identifier for the request',
        'run_id': 'The canonical Run identifier for the request',
        'extra_run_id': 'The canonical Extra Run identifier for the request',
        'msg': 'Log message'
    })
    @log_api.response(400, 'Post data Validation Error', model_validation_response)
    def post(self):
        '''Upload Job Run Logs'''
        parser = reqparse.RequestParser()
        parser.add_argument('job_id', required=True, type=str, default='',
                            help="The Job canonical identifier for the request")
        parser.add_argument('run_id', required=True, type=str, default='',
                            help="The Run canonical identifier for the request")
        parser.add_argument('extra_run_id', required=True, type=str, default='',
                            help="The extra identifier for the run")
        parser.add_argument('msg', required=True, type=str, default='', help="The log msg ")
        args = parser.parse_args()
        fn = _params_to_filename(
            job_id=args.job_id,
            run_id=args.run_id,
            extra_run_id=args.extra_run_id
        )
        ret = {
            "error_msg": "",
            "has_error": False

        }
        status_code = 500
        try:
            with open(fn, "a") as myfile:
                myfile.write(args.msg)
            status_code = 200
        except Exception as e:
            error_type = type(e).__name__
            logger.debug(f"Got error {error_type} {e} {traceback.print_stack(limit=10)}")
            ret['error_msg'] = f"{error_type}: {e}"
            ret['has_error'] = True

        return ret, status_code

    @log_api.doc(params={
        'job_id': 'The canonical Job identifier for the request',
        'run_id': 'The canonical Run identifier for the request',
        'extra_run_id': 'The canonical Extra Run identifier for the request'
    })
    @log_api.response(400, 'Post data Validation Error', model_validation_response)
    def delete(self):
        '''Delete Job Run Logs'''
        parser = reqparse.RequestParser()
        parser.add_argument('job_id', required=True, type=str, default='',
                            help="The Job canonical identifier for the request")
        parser.add_argument('run_id', required=True, type=str, default='',
                            help="The Run canonical identifier for the request")
        parser.add_argument('extra_run_id', required=True, type=str, default='',
                            help="The extra identifier for the run")
        args = parser.parse_args()
        fn = _params_to_filename(
            job_id=args.job_id,
            run_id=args.run_id,
            extra_run_id=args.extra_run_id
        )
        ret = {
            "error_msg": "",
            "has_error": False
        }
        status_code = 500
        try:
            if os.path.exists(fn):
                os.remove(fn)
            status_code = 200
        except Exception as e:
            error_type = type(e).__name__
            logger.debug(f"Got error {error_type} {e} {traceback.print_stack(limit=10)}")
            ret['error_msg'] = f"{error_type}: {e}"
            ret['has_error'] = True

        return ret, status_code
