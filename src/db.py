from asyncpg import create_pool

DSN = "postgresql://{user}:{password}@{host}:{port}/{database}"

async def init_pg(app):
    config = app['config']['postgres']
    pool = await create_pool(DSN.format(**config), min_size = 20, max_size = 20)
    app['db_pool'] = pool

async def close_pg(app):
    await app['db_pool'].close()