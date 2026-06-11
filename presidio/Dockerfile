FROM python:3.14-slim@sha256:d7a925f9eb9639a93e455b9f12c167569358818c0f62b51b88edbc8fcf34c421

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

CMD pipenv run python -m server --host 0.0.0.0
