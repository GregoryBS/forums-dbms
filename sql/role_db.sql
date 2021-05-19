create role anna with login superuser password 'yoh';
CREATE DATABASE forums
    WITH 
    OWNER = anna
    ENCODING = 'UTF8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1;
