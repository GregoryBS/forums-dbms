create table users (
    nick text not null primary key,
    full text not null,
    email text not null,
    about text
);

create table forums (
    slug text not null primary key,
    title text not null,
    author text not null,
    threads int,
    posts bigint,
    constraint to_user foreign key (author) references users(nick) on delete cascade
);

create table threads (
    id serial not null primary key,
    title text not null,
    author text not null,
    forum text not null,
    message text not null,
    slug text not null unique,
    created timestamp
    votes int,
    constraint to_user foreign key (author) references users(nick) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade
);

create table posts (
    id bigserial not null primary key,
    parent bigint not null default 0,
    author text not null,
    forum text not null,
    thread int not null,
    message text not null,
    created timestamp,
    constraint to_user foreign key (author) references users(nick) on delete cascade,
    constraint to_forum foreign key (forum) references forums(slug) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);

create table votes (
    author text not null,
    thread int not null,
    value smallint,
    unique (author, thread),
    constraint to_user foreign key (author) references users(nick) on delete cascade,
    constraint to_thread foreign key (thread) references threads(id) on delete cascade
);
