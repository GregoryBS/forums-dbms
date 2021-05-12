from aiohttp import web

from src import db
from src.settings import config
from src.routes import setup_routes

def main():
    app = web.Application()
    setup_routes(app)
    app['config'] = config
    app.on_startup.append(db.init_pg)
    app.on_cleanup.append(db.close_pg)
    web.run_app(app, port=config['app']['port'])


main()
