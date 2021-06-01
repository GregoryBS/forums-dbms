FROM postgres:latest

RUN apt-get update && apt-get install -y python3 python3-pip
RUN pip3 install aiohttp asyncpg pyyaml

ENV TZ=Russia/Moscow
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENV POSTGRES_DB forums
ENV POSTGRES_USER postgres
ENV POSTGRES_PASSWORD password

USER $POSTGRES_USER

WORKDIR /app
COPY . .

RUN pg_createcluster 13 main &&\
    service postgresql start &&\
    psql -U $POSTGRES_USER -f sql/role_db.sql &&\
    psql -U $POSTGRES_USER -d $POSTGRES_DB -f sql/tables.sql &&\
    service postgresql stop

VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

EXPOSE 5000

CMD service postgresql start && python3 main.py
