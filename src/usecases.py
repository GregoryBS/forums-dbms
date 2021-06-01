from asyncpg.exceptions import UniqueViolationError, ForeignKeyViolationError

from datetime import datetime

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

            created = datetime.now().strftime('%Y-%m-%dT%H:%M:%S.%fZ')
            if form.get('created'):
                created = form['created']
            thread = await conn.fetchrow("insert into threads values(default, $1, $2, $3, $4, $5, $6, 0) returning *;", 
                                        form['title'], data[0]['slug'], data[1]['slug'], form['message'], form.get('slug'), created)
            thread = dict(thread)
            return thread, 201

        except:
            thread = await conn.fetchrow("select id, title, author, votes, message, forum, slug, created from threads where slug = $1;", form['slug'])
            thread = dict(thread)
            return thread, 409

async def create_post(app, ident, posts):
    async with app['db_pool'].acquire() as conn:
        async with conn.transaction():
            try:
                thread = await conn.fetchrow("select id, forum from threads where {:s} = $1;".format(ident['name']), ident['value'])
                if thread is None:
                    error = {'message': 'thread not found'}
                    return error, 404

                created = datetime.now().strftime('%Y-%m-%dT%H:%M:%S.%fZ')
                query = await conn.prepare("insert into posts values(default, $1, $2, $3, $4, $5, $6, false, null) " + \
                    "returning id, parent, author, forum, thread, message, created, edit;")
                for i in range(len(posts)):
                    posts[i] = await query.fetchrow(posts[i].get('parent', 0), posts[i]['author'], 
                                                    thread['forum'], thread['id'], posts[i]['message'], created)
                    posts[i] = dict(posts[i])
                return posts, 201

            except ForeignKeyViolationError:
                error = {'message': 'author not found'}
                return error, 404

            except:
                error = {'message': 'cannot create posts'}
                return error, 409

async def get_thread(app, ident):
    async with app['db_pool'].acquire() as conn:
        try:
            thread = await conn.fetchrow("select id, forum, title, author, created, message, slug, votes from threads where {:s} = $1;".
                                         format(ident['name']), ident['value'])
            thread = dict(thread)
            return thread, 200

        except:
            error = {'message': 'thread not found'}
            return error, 404

async def forum_threads(app, slug, limit, since, desc):
    counter = 2
    fields = []
    query = "select id, forum, title, author, created, message, slug, votes from threads where forum = $1 "
    if since:
        if desc == 'true':
            query += "and created <= ${:d} ".format(counter)
        else:
            query += "and created >= ${:d} ".format(counter)

        fields.append(since)
        counter += 1

    query += "order by created "   
    if desc == 'true':
        query += "desc "
    query += "limit ${:d};".format(counter)
    fields.append(limit)

    async with app['db_pool'].acquire() as conn:
        forum = await conn.fetchrow("select slug from forums where slug = $1;", slug)
        if forum is None:
            error = {'message': 'forum not found'}
            return error, 404

        threads = await conn.fetch(query, slug, *fields)
        threads = list(map(dict, threads))
        return threads, 200

async def clear(app):
    async with app['db_pool'].acquire() as conn:
        try:
            await conn.execute("truncate users cascade;")
            return 200

        except Exception as e:
            print("unexpected exception while clearing db: ", e)
            return 500

async def status(app):
    async with app['db_pool'].acquire() as conn:
        try:
            data = await conn.fetch("select count(nickname) from users union all select count(slug) from forums " + \
                "union all select count(id) from threads union all select count(id) from posts;")
            response = {}
            labels = ['user', 'forum', 'thread', 'post']
            for i in range(len(data)):
                response[labels[i]] = data[i].get('count')
            return response, 200

        except Exception as e:
            print("unexpected exception while getting status of db: ", e)
            return None, 500

async def new_vote(app, ident, vote):
    async with app['db_pool'].acquire() as conn:
        try:
            thread = await conn.fetchrow("select id from threads where {:s} = $1;".format(ident['name']), ident['value'])
            if thread is None:
                error = {'message': 'thread not found'}
                return error, 404
                
            await conn.execute("insert into votes values($1, $2, $3);", vote['nickname'], thread['id'], vote['voice'])
            return await get_thread(app, ident)

        except UniqueViolationError:
            await conn.execute("update votes set value = $1 where author = $2 and thread = $3;", vote['voice'], vote['nickname'], thread['id'])
            return await get_thread(app, ident)

        except ForeignKeyViolationError:
            error = {'message': 'user not found'}
            return error, 404

async def update_thread(app, ident, form):
    query = "update threads set "
    counter = 1
    fields = []
    if form.get('title'):
        query += ("title = ${:d}".format(counter))
        counter += 1
        fields.append(form['title'])
    if form.get('message'):
        query += "," if counter > 1 else ""
        query += ("message = ${:d}".format(counter))
        counter += 1
        fields.append(form['message'])
    query += (" where {:s} = ${:d} returning *;".format(ident['name'], counter))
    fields.append(ident['value'])

    async with app['db_pool'].acquire() as conn:
        try:
            thread = await conn.fetchrow(query, *fields)
            if thread is None:
                error = {'message': 'thread not found'}
                return error, 404

            thread = dict(thread)
            return thread, 200

        except:
            error = {'message': 'thread cannot be updated'}
            return error, 409

async def thread_posts(app, ident, limit, since, sort, desc):
    counter = 2
    fields = []
    query = "select id, parent, author, forum, thread, message, created, edit from posts where thread = $1 "

    if sort == 'flat':
        if since:
            if desc == 'true':
                query += "and id < ${:d} ".format(counter)
            else:
                query += "and id > ${:d} ".format(counter)

            fields.append(since)
            counter += 1
        if desc == 'true':
            query += "order by created desc, id desc "   
        else:
            query += "order by created, id "   
        query += "limit ${:d};".format(counter)

    elif sort == 'tree':
        if since:
            if desc == 'true':
                query += "and path < (select path from posts where id = ${:d}) ".format(counter)
            else:
                query += "and path > (select path from posts where id = ${:d}) ".format(counter)

            fields.append(since)
            counter += 1
        query += "order by path " 
        if desc == 'true':
            query += "desc "       
        query += "limit ${:d};".format(counter)

    elif sort == 'parent_tree':
        query += "and path[1] in (select id from posts where thread = $1 and parent = 0 "
        if since:
            if desc == 'true':
                query += "and path[1] < (select path[1] from posts where id = ${:d}) ".format(counter)
            else:
                query += "and path[1] > (select path[1] from posts where id = ${:d}) ".format(counter)

            fields.append(since)
            counter += 1

        if desc == 'true':
            query += "order by id desc limit ${:d}) order by path[1] desc, path;".format(counter)
        else:
            query += "order by id limit ${:d}) order by path[1], path;".format(counter)

    fields.append(limit)

    async with app['db_pool'].acquire() as conn:
        thread = await conn.fetchrow("select id from threads where {:s} = $1;".format(ident['name']), ident['value'])
        if thread is None:
            error = {'message': 'thread not found'}
            return error, 404

        posts = await conn.fetch(query, thread['id'], *fields)
        posts = list(map(dict, posts))
        return posts, 200

async def forum_users(app, slug, limit, since, desc):
    query = "select u.nickname, u.fullname, u.email, u.about from users as u join posts as p on u.nickname = p.author where p.forum = $1 " + \
        "union select u.nickname, u.fullname, u.email, u.about from users as u join threads as t on u.nickname = t.author where t.forum = $1;"

    async with app['db_pool'].acquire() as conn:
        forum = await conn.fetchrow("select slug from forums where slug = $1;", slug)
        if forum is None:
            error = {'message': 'forum not found'}
            return error, 404

        users = await conn.fetch(query, slug)
        users = list(map(dict, users))
        if since:
            since = since.lower()
            if desc == 'true':
                users = list(filter(lambda x: x['nickname'].lower() < since, users))
            else:
                users = list(filter(lambda x: x['nickname'].lower() > since, users))
        users.sort(key = lambda x: x['nickname'].lower(), reverse = (desc == 'true'))
        return users[:limit], 200

async def update_post(app, id, form):
    post, status = await get_post(app, id, [])
    if status != 200:
        return post, status
    post = post['post']
    if form.get('message') is None or form.get('message') == post['message']:
        return post, 200

    async with app['db_pool'].acquire() as conn:
        try:
            await conn.execute("update posts set message = $1, edit = true where id = $2;", form['message'], id)

            post['message'] = form['message']
            post['isEdited'] = True
            return post, 200

        except:
            error = {'message': 'thread cannot be updated'}
            return error, 409

async def get_post(app, id, related):
    async with app['db_pool'].acquire() as conn:
        data = {}
        post = await conn.fetchrow("select id, parent, author, forum, thread, message, created, edit from posts where id = $1;", id)
        if post is None:
            error = {'message': 'post not found'}
            return error, 404

        post = dict(post)
        post['isEdited'] = post.pop('edit')
        data['post'] = post
        if 'forum' in related:
            forum, status = await get_forum(app, post['forum'])
            if status != 200:
                return forum, status
            data['forum'] = forum
        if 'thread' in related:
            thread, status = await get_thread(app, {'name': 'id', 'value': post['thread']})
            if status != 200:
                return thread, status
            data['thread'] = thread
        if 'user' in related:
            author, status = await get_profile(app, post['author'])
            if status != 200:
                return author, status
            data['author'] = author
        
        return data, 200
