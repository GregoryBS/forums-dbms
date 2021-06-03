create role anna with login superuser password 'yoh';
CREATE DATABASE forums
    WITH 
    OWNER = anna
    TEMPLATE = 'template0'
    ENCODING = 'UTF8'
    LC_COLLATE = 'C'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;
