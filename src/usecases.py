from asyncpg.exceptions import UniqueViolationError, ForeignKeyViolationError

from datetime import datetime

def format_datetime(x):
    x['created'] = x['created'].strftime('%Y-%m-%dT%H:%M:%S.%fZ')
    return x

async def signup(app, nick, form):
    async with app['db_pool'].acquire() as conn:
        try:
            await conn.execute("insert into users values($1, $2, $3, $4);", nick, form['fullname'], form['email'], form['about'])
            form['nickname'] = nick
            return form, 201

        except:
            users = await conn.fetch("select nickname, fullname, email, about from users where nickname = $1 or email = $2;", nick, form['email'])
            return list(map(dict, users)), 409

async def get_profile(app, nick):
    async with app['db_pool'].acquire() as conn:
        try:
            user = await conn.fetchrow("select nickname, fullname, email, about from users where nickname = $1;", nick)
            return dict(user), 200

        except:
            error = {'message': 'user not found'}
            return error, 404

async def update_profile(app, nick, form):
    query = "update users set "
    counter = 1
    fields = []
    if form.get('fullname'):
        query += ("fullname = ${:d}".format(counter))
        counter += 1
        fields.append(form['fullname'])
    if form.get('email'):
        query += "," if counter > 1 else ""
        query += ("email = ${:d}".format(counter))
        counter += 1
        fields.append(form['email'])
    if form.get('about'):
        query += "," if counter > 1 else ""
        query += ("about = ${:d}".format(counter))
        counter += 1
        fields.append(form['about'])
    query += (" where nickname = ${:d} returning *;".format(counter))
    fields.append(nick)

    async with app['db_pool'].acquire() as conn:
        try:
            user = await conn.fetchrow(query, *fields)
            if user is None:
                error = {'message': 'user not found'}
                return error, 404

            return dict(user), 200

        except:
            error = {'message': 'user cannot be updated'}
            return error, 409

async def create_forum(app, form):
     async with app['db_pool'].acquire() as conn:
        try:
            user = await conn.fetchrow("select nickname from users where nickname = $1;", form['user'])
            if user is None:
                error = {'message': 'user not found'}
                return error, 404

            forum = await conn.fetchrow("insert into forums values($1, $2, $3, 0, 0) returning slug, title, author as user, posts, threads;", 
                                        form['slug'], form['title'], user['nickname'])
            return dict(forum), 201

        except:
            forum = await conn.fetchrow("select slug, title, author as user, threads, posts from forums where slug = $1;", form['slug'])
            return dict(forum), 409

async def get_forum(app, slug):
    async with app['db_pool'].acquire() as conn:
        try:
            forum = await conn.fetchrow("select slug, title, author as user, threads, posts from forums where slug = $1;", slug)
            return dict(forum), 200

        except:
            error = {'message': 'forum not found'}
            return error, 404

async def create_thread(app, slug, form):
    async with app['db_pool'].acquire() as conn:
        try:
            data = await conn.fetch("select nickname as slug from users where nickname = $1 union all select slug from forums where slug = $2;", 
                                    form['author'], slug)
            data = list(map(dict, data))
            if len(data) < 2:
                error = {'message': 'user or forum not found'}
                return error, 404

            created = datetime.now()
            if form.get('created'):
                created = datetime.strptime(form['created'], '%Y-%m-%dT%H:%M:%S.%fZ')
            thread = await conn.fetchrow("insert into threads values(default, $1, $2, $3, $4, $5, $6, 0) returning *;", 
                                        form['title'], data[0]['slug'], data[1]['slug'], form['message'], form.get('slug'), created)
            thread = dict(thread)
            format_datetime(thread)
            return thread, 201

        except:
            thread = await conn.fetchrow("select id, title, author, votes, message, forum, slug, created from threads where slug = $1;", form['slug'])
            thread = dict(thread)
            format_datetime(thread)
            return thread, 409

async def create_post(app, ident, posts):
    async with app['db_pool'].acquire() as conn:
        async with conn.transaction():
            try:
                thread = await conn.fetchrow("select id, forum from threads where {:s} = $1;".format(ident['name']), ident['value'])
                if thread is None:
                    error = {'message': 'thread not found'}
                    return error, 404

                created = datetime.now()
                query = await conn.prepare("insert into posts values(default, $1, $2, $3, $4, $5, $6, false) returning *;")
                for i in range(len(posts)):
                    posts[i] = await query.fetchrow(posts[i].get('parent', 0), posts[i]['author'], 
                                                    thread['forum'], thread['id'], posts[i]['message'], created)
                    posts[i] = dict(posts[i])
                    format_datetime(posts[i])
                return posts, 201

            except:
                error = {'message': 'cannot create posts'}
                return error, 409

async def get_thread(app, ident):
    async with app['db_pool'].acquire() as conn:
        try:
            thread = await conn.fetchrow("select id, forum, title, author, created, message, slug, votes from threads where {:s} = $1;".
                                         format(ident['name']), ident['value'])
            thread = dict(thread)
            format_datetime(thread)
            return thread, 200

        except:
            error = {'message': 'thread not found'}
            return error, 404

async def forum_threads(app, slug, limit, since, desc):
    async with app['db_pool'].acquire() as conn:
        forum = await conn.fetchrow("select slug from forums where slug = $1;", slug)
        if forum is None:
            error = {'message': 'forum not found'}
            return error, 404

        counter = 2
        fields = []
        query = "select id, forum, title, author, created, message, slug, votes from threads where forum = $1 "
        if since:
            if desc == 'true':
                query += "and created <= ${:d} ".format(counter)
            else:
                query += "and created >= ${:d} ".format(counter)

            since = datetime.strptime(since, '%Y-%m-%dT%H:%M:%S.%fZ')
            fields.append(since)
            counter += 1

        query += "order by created "   
        if desc == 'true':
            query += "desc "
        query += "limit ${:d};".format(counter)
        fields.append(limit)

        threads = await conn.fetch(query, slug, *fields)
        threads = list(map(dict, threads))
        threads = list(map(format_datetime, threads))
        return threads, 200
