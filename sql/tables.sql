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
    slug citext unique,
    created timestamp with time zone,
    votes int,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade
);

create table posts (
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
for each row
execute function update_forum_threads();

create function posts_path()
returns trigger as $$
declare
    r record;
begin 
    if NEW.parent = 0 then
        NEW.path = array[NEW.id];
    else
        select array_append(path, NEW.id) as pth, thread from posts where id = NEW.parent into r;
        if r.thread != NEW.thread or NEW.parent not in (select id from posts) then
            return null;
        end if;
        NEW.path = r.pth;
    end if;
    return NEW;
end;
$$ language plpgsql;

create trigger new_post_path before insert
on posts
for each row
execute function posts_path();

create function update_forum_posts()
returns trigger as $$
begin 
    update forums set posts = posts + 1 where slug = NEW.forum;
    return NEW;

end;
$$ language plpgsql;

create trigger new_forum_post after insert
on posts
for each row
execute function update_forum_posts();

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
