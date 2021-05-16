
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
