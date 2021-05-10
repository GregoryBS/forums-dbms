FROM postgres:latest

RUN apt-get update && apt-get install -y python3 python3-pip
RUN pip3 install aiohttp asyncpg

ENV POSTGRES_DB forums
ENV POSTGRES_USER postgres
ENV POSTGRES_PASSWORD password

USER $POSTGRES_USER

RUN pg_createcluster 13 main &&\
    service postgresql start &&\
    psql -U $POSTGRES_USER -f role_db.sql &&\
    psql -U $POSTGRES_USER -d $POSTGRES_DB -f tables.sql &&\
    service postgresql stop

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

EXPOSE 5000
WORKDIR /app
COPY . .

CMD service postgresql start && python3 main.py
