FROM golang:latest as build

WORKDIR /app

COPY main.go main.go
COPY go.mod go.mod

RUN go mod tidy
RUN go build -o main main.go


FROM postgres:latest

ENV POSTGRES_DB forums
ENV POSTGRES_USER postgres
ENV POSTGRES_PASSWORD password
ENV PGVER 13

USER $POSTGRES_USER

WORKDIR /app

COPY config.json config.json
COPY sql/ sql/

RUN pg_createcluster $PGVER main &&\
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

COPY --from=build /app/main .

CMD service postgresql start && ./main
