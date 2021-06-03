FROM postgres:latest

RUN apt-get update && apt-get install -y python3 python3-pip
RUN pip3 install aiohttp asyncpg pyyaml

ENV POSTGRES_DB forums
ENV POSTGRES_USER postgres
ENV POSTGRES_PASSWORD password
ENV PGVER 13

USER $POSTGRES_USER

WORKDIR /app
COPY . .

RUN pg_createcluster 13 main &&\
    service postgresql start &&\
    psql -U $POSTGRES_USER -f sql/role_db.sql &&\
    psql -U $POSTGRES_USER -d $POSTGRES_DB -f sql/tables.sql &&\
    service postgresql stop

RUN echo "synchronous_commit=off" >> /etc/postgresql/$PGVER/main/postgresql.conf &&\
    echo "fsync=off" >> /etc/postgresql/$PGVER/main/postgresql.conf &&\
    echo "work_mem=32MB" >> /etc/postgresql/$PGVER/main/postgresql.conf &&\
    echo "maintenance_work_mem=256MB" >> /etc/postgresql/$PGVER/main/postgresql.conf &&\
    echo "shared_buffers=768MB" >> /etc/postgresql/$PGVER/main/postgresql.conf &&\
    echo "full_page_writes=off" >> /etc/postgresql/$PGVER/main/postgresql.conf &&\
    echo "unix_socket_directories = '/var/run/postgresql'" >> /etc/postgresql/$PGVER/main/postgresql.conf

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

EXPOSE 5000

CMD service postgresql start && python3 main.py
