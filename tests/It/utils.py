from typing import Iterator, Collection

import mysql.connector
import config
import logging


def progressbar(iterable: Collection, fill_char: str = '█', end: str = "\r") -> Iterator:
    """
    Progress Bar Printing Function
    :param iterable: the collection we iterate
    :param fill_char: fill character for progressbar
    :param end: line end character
    :return: iterated item
    """
    total = len(iterable)
    prefix = 'Progress [ ]:'
    suffix = 'Complete'
    bar_length: int = 100

    def print_progress_bar(iteration):
        percent = "{0:.1f}".format(100 * (iteration / float(total)))
        filled_length = int(bar_length * iteration // total)
        bar = fill_char * filled_length + '-' * (bar_length - filled_length)
        print(f'\r{prefix} |{bar}| {percent}% {suffix}', end=end)

    # Init
    print_progress_bar(0)
    # Update Progress Bar
    for i, item in enumerate(iterable):
        prefix = f"Progress [{item['id']}]:"
        yield item
        print_progress_bar(i + 1)
    # New Line on Complete
    print()


def query(sql, params=()):
    mysql_config = {
        'user': config.config.get('MYSQL_DATABASE_USER'),
        'database': config.config.get('MYSQL_DATABASE_DB'),
        'password': config.config.get('MYSQL_DATABASE_PASSWORD'),
        'host': config.config.get('MYSQL_DATABASE_HOST'),
        'port': config.config.get('MYSQL_DATABASE_PORT')
    }

    cnx = mysql.connector.connect(**mysql_config)
    response = []
    cursor = cnx.cursor()
    try:
        cursor.execute(sql, params)
        if 'UPDATE' in sql or 'INSERT' in sql or 'DELETE' in sql or 'ALTER' in sql or 'TRUNCATE' in sql:
            cnx.commit()
        result = []
        columns = cursor.description
        for value in cursor.fetchall():
            tmp = {}
            for (index, column) in enumerate(value):
                tmp[columns[index][0]] = column
            result.append(tmp)
        response = result
    except mysql.connector.Error as error:
        logging.warning(f"Can't perform {sql} got {error}")
    return response


def truncate(initial_number: int):
    query(f"TRUNCATE TABLE {config.config.get('MYSQL_DATABASE_TABLE')}")
    logging.info("Truncating table")
    query(f"ALTER TABLE {config.config.get('MYSQL_DATABASE_TABLE')}  AUTO_INCREMENT={initial_number}")
    logging.info(f"Resetting autoincrement -> {initial_number}")
