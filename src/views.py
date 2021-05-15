from aiohttp import web

async def index(request):
    return web.Response(text='Hello Aiohttp!')

async def signup(request):
    nick = request.match_info['nick']
    data = await request.json()
    pool = request.app['db_pool']
    async with pool.acquire() as conn:
        try:
            data['nickname'] = nick
            await conn.execute("insert into users values($1, $2, $3, $4);", nick, data['fullname'], data['email'], data['about'])
            return web.json_response(data, status = 201)
        except:
            users = await conn.fetch("select nickname, fullname, email, about from users where nickname = $1 or email = $2;", nick, data['email'])
            return web.json_response(list(map(dict, users)), status = 409)

async def get_profile(request):
    nick = request.match_info['nick']
    pool = request.app['db_pool']
    async with pool.acquire() as conn:
        try:
            user = await conn.fetchrow("select nickname, fullname, email, about from users where nickname = $1;", nick)
            return web.json_response(dict(user), status = 200)
        except:
            error = {'message': 'user not found'}
            return web.json_response(error, status = 404)

async def update_profile(request):
    nick = request.match_info['nick']
    data = await request.json()
    pool = request.app['db_pool']
    async with pool.acquire() as conn:
        try:
            result = await conn.execute("update users set fullname = $1, email = $2, about = $3 where nickname = $4;", data['fullname'], data['email'], data['about'], nick)
            if int(result.split(' ')[-1]) == 0:
                error = {'message': 'user not found'}
                return web.json_response(error, status = 404)
            data['nickname'] = nick
            return web.json_response(data, status = 200)
        except:
            error = {'message': 'user cannot be updated'}
            return web.json_response(error, status = 409)