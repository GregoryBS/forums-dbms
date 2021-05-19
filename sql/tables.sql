create extension if not exists citext; 

create table users (
    nickname citext primary key,
    fullname text not null,
    email citext not null unique,
    about text
);

create table forums (
    slug citext primary key,
    title text not null,
    author citext not null,
    threads int,
    posts bigint,
    constraint to_user foreign key (author) references users(nickname) on delete cascade
);

create table threads (
    id serial primary key,
    title text not null,
    author citext not null,
    forum citext not null,
    message text not null,
    slug text unique,
    created timestamp,
    votes int,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade
);

create table posts (
    id bigserial primary key,
    parent bigint default 0 references posts(id),
    author citext not null,
    forum citext not null,
    thread int not null,
    message text not null,
    created timestamp,
    edit boolean,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);

create table votes (
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
execute function update_forum_threads();

create function update_forum_posts()
returns trigger as $$
begin 
    update forums set posts = posts + 1 where slug = NEW.forum;
    return NEW;
end;
$$ language plpgsql;

create trigger new_forum_post after insert
on posts
execute function update_forum_posts();
