from asyncpg import create_pool

DSN = "postgresql://{user}:{password}@{host}:{port}/{database}"

async def init_pg(app):
    config = app['config']['postgres']
    pool = create_pool(DSN.format(**config))
    app['db_pool'] = pool

async def close_pg(app):
    app['db_pool'].close()