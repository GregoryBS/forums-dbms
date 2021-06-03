create extension if not exists citext; 

create unlogged table users (
    nickname citext primary key,
    fullname text not null,
    email citext not null unique,
    about text
);

create unlogged table forums (
    slug citext primary key,
    title text not null,
    author citext not null,
    threads int,
    posts bigint,
    constraint to_user foreign key (author) references users(nickname) on delete cascade
);

create unlogged table threads (
    id serial primary key,
    title text not null,
    author citext not null,
    forum citext not null,
    message text not null,
    slug citext unique,
    created timestamp with time zone,
    votes int,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade
);

create unlogged table posts (
    id bigserial primary key,
    parent bigint not null,
    author citext not null,
    forum citext not null,
    thread int not null,
    message text not null,
    created timestamp with time zone,
    edit boolean,
    path bigint array,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);

create unlogged table votes (
    author citext,
    thread int,
    value smallint,
    primary key (author, thread),
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);

create function update_forum_threads()
returns trigger as $$
begin 
    update forums set threads = threads + 1 where slug = NEW.forum;
    return NEW;
end;
$$ language plpgsql;

create trigger new_forum_thread after insert
on threads
for each row
execute function update_forum_threads();

create function posts_path()
returns trigger as $$
begin 
    NEW.path = array_append(NEW.path, NEW.id);
    return NEW;
end;
$$ language plpgsql;

create trigger new_post_path before insert
on posts
for each row
execute function posts_path();

create function update_thread_votes()
returns trigger as $$
begin 
    if TG_OP = 'INSERT' then
        update threads set votes = votes + NEW.value where id = NEW.thread;
        return NEW;
    end if;
    if OLD.value != NEW.value then 
        update threads set votes = votes + 2 * NEW.value where id = NEW.thread;
        return NEW;
    end if;
    return null;
end;
$$ language plpgsql;

create trigger new_thread_vote after insert or update
on votes
for each row
execute function update_thread_votes();

create index hash_user_key ON users using hash (nickname);
create index thread_keys ON threads(slug, id);
create index hash_thread_id ON threads using hash (id);
create index hash_thread_slug ON threads using hash (slug);
create index thread_forum_author ON threads(forum, author);
create index post_keys ON posts(id, path, thread);
create index post_path ON posts(path);
create index post_forum_author ON posts(forum, author);
create index post_thread_parent ON posts(parent, thread);
