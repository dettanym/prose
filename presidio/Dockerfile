FROM python:3.12-slim@sha256:db7e9284d53f7b827c58a6239b9d2907c33250215823b1cdb7d1e983e70dafa5

ARG NLP_CONF_FILE=server/conf/default.yaml

ENV PORT=3000
ENV PIPENV_VENV_IN_PROJECT=1
ENV PIP_NO_CACHE_DIR=1

EXPOSE ${PORT}
WORKDIR /app

COPY ./Pipfile* /app/
RUN \
    pip install pipenv \
    && pipenv sync

# install nlp models specified in conf/default.yaml
COPY ./scripts/ /app/scripts/
COPY ${NLP_CONF_FILE} /app/${NLP_CONF_FILE}

RUN pipenv run python scripts/install_nlp_models.py --conf_file /app/${NLP_CONF_FILE}

COPY server/ /app/server/

CMD pipenv run python /app/server --host 0.0.0.0
