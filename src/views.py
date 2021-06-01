from aiohttp import web

from . import usecases

def get_slug_or_id(request):
    slug_or_id = request.match_info['slug_or_id']
    try:
        value = int(slug_or_id)
        slug_or_id = {'name': 'id', 'value': value}
    except:
        slug_or_id = {'name': 'slug', 'value': slug_or_id}
    return slug_or_id

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
    slug_or_id = get_slug_or_id(request)
    data = await request.json()
    data, status = await usecases.create_post(request.app, slug_or_id, data)
    return web.json_response(data, status = status)

async def get_thread(request):
    slug_or_id = get_slug_or_id(request)
    data, status = await usecases.get_thread(request.app, slug_or_id)
    return web.json_response(data, status = status)

async def get_forum_threads(request):
    slug = request.match_info['slug']
    limit = int(request.query.get('limit', 100))
    since = request.query.get('since')
    desc = request.query.get('desc', 'false')
    data, status = await usecases.forum_threads(request.app, slug, limit, since, desc)
    return web.json_response(data, status = status)

async def clear(request):
    status = await usecases.clear(request.app)
    return web.json_response(None, status = status)

async def get_status(request):
    data, status = await usecases.status(request.app)
    return web.json_response(data, status = status)

async def thread_vote(request):
    slug_or_id = get_slug_or_id(request)
    data = await request.json()
    data, status = await usecases.new_vote(request.app, slug_or_id, data)
    return web.json_response(data, status = status)

async def update_thread(request):
    slug_or_id = get_slug_or_id(request)
    data = await request.json()
    if len(data) != 0:
        data, status = await usecases.update_thread(request.app, slug_or_id, data)
    else:
        data, status = await usecases.get_thread(request.app, slug_or_id)
    return web.json_response(data, status = status)

async def get_thread_posts(request):
    slug_or_id = get_slug_or_id(request)
    limit = int(request.query.get('limit', 100))
    since = int(request.query.get('since', 0))
    sort = request.query.get('sort', 'flat')
    desc = request.query.get('desc', 'false')
    data, status = await usecases.thread_posts(request.app, slug_or_id, limit, since, sort, desc)
    return web.json_response(data, status = status)

async def get_forum_users(request):
    slug = request.match_info['slug']
    limit = int(request.query.get('limit', 100))
    since = request.query.get('since')
    desc = request.query.get('desc', 'false')
    data, status = await usecases.forum_users(request.app, slug, limit, since, desc)
    return web.json_response(data, status = status)

async def update_post(request):
    id = int(request.match_info['id'])
    data = await request.json()
    data, status = await usecases.update_post(request.app, id, data)
    return web.json_response(data, status = status)

async def get_post(request):
    id = int(request.match_info['id'])
    related = request.query.get('related', '').split(',')
    data, status = await usecases.get_post(request.app, id, related)
    return web.json_response(data, status = status)
