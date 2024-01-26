FROM python:3.9-slim

ARG NLP_CONF_FILE=conf/default.yaml
ENV PIPENV_VENV_IN_PROJECT=1
ENV PIP_NO_CACHE_DIR=1
WORKDIR /app

COPY ./Pipfile* /app/
RUN pip install pipenv \
  && pipenv sync
# install nlp models specified in conf/default.yaml
COPY ./install_nlp_models.py /app/
COPY ${NLP_CONF_FILE} /app/${NLP_CONF_FILE}
RUN pipenv run python install_nlp_models.py --conf_file /app/${NLP_CONF_FILE}

COPY . /app/
EXPOSE ${PORT}
CMD pipenv run python /app/app.py --host 0.0.0.0
