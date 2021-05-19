from aiohttp import web

from . import usecases

async def index(request):
    return web.Response(text='Hello Aiohttp!')

async def signup(request):
    nick = request.match_info['nick']
    data = await request.json()
    data, status = await usecases.signup(request.app, nick, data)
    return web.json_response(data, status = status)

async def get_profile(request):
    nick = request.match_info['nick']
    data, status = await usecases.get_profile(request.app, nick)
    return web.json_response(data, status = status)

async def update_profile(request):
    nick = request.match_info['nick']
    data = await request.json()
    if len(data) != 0:
        data, status = await usecases.update_profile(request.app, nick, data)
    else:
        data, status = await usecases.get_profile(request.app, nick)
    return web.json_response(data, status = status)

async def create_forum(request):
    data = await request.json()
    data, status = await usecases.create_forum(request.app, data)
    return web.json_response(data, status = status)
        
async def get_forum(request):
    slug = request.match_info['slug']
    data, status = await usecases.get_forum(request.app, slug)
    return web.json_response(data, status = status)

async def create_thread(request):
    slug = request.match_info['slug']
    data = await request.json()
    data, status = await usecases.create_thread(request.app, slug, data)
    return web.json_response(data, status = status)

async def create_post(request):
    slug_or_id = request.match_info['slug_or_id']
    data = await request.json()
    data, status = await usecases.create_post(request.app, slug_or_id, data)
    return web.json_response(data, status = status)

async def get_thread(request):
    slug_or_id = request.match_info['slug_or_id']
    data, status = await usecases.get_thread(request.app, slug_or_id)
    return web.json_response(data, status = status)

async def get_forum_threads(request):
    slug = request.match_info['slug']
    limit = int(request.query.get('limit', 100))
    since = request.query.get('since')
    desc = request.query.get('desc', 'false')
    data, status = await usecases.forum_threads(request.app, slug, limit, since, desc)
    return web.json_response(data, status = status)
