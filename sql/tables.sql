create extension if not exists citext; 

create table users (
    nickname citext not null primary key,
    fullname text not null,
    email citext not null unique,
    about text
);

create table forums (
    slug citext not null primary key,
    title text not null,
    author citext not null,
    threads int,
    posts bigint,
    constraint to_user foreign key (author) references users(nickname) on delete cascade
);

create table threads (
    id serial not null primary key,
    title text not null,
    author citext not null,
    forum citext not null,
    message text not null,
    slug text not null unique,
    created timestamp,
    votes int,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade
);

create table posts (
    id bigserial not null primary key,
    parent bigint not null default 0,
    author citext not null,
    forum citext not null,
    thread int not null,
    message text not null,
    created timestamp,
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);

create table votes (
    author citext not null,
    thread int not null,
    value smallint,
    unique (author, thread),
    constraint to_user foreign key (author) references users(nickname) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);
