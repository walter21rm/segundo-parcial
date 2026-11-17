# syntax=docker/dockerfile:1

FROM python:3.12.8-slim-bookworm AS build
WORKDIR /src
RUN --mount=type=bind,source=requirements.txt,target=requirements.txt \
    --mount=type=cache,target=/root/.cache/pip \
    pip install --prefix=/install -r requirements.txt

FROM python:3.12.8-slim-bookworm AS runtime
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1
RUN useradd --system --uid 10001 --create-home --shell /usr/sbin/nologin appuser
COPY --from=build --link /install /usr/local
WORKDIR /app
COPY --link app ./app
USER appuser
EXPOSE 8092
ENTRYPOINT ["python", "-m", "app.main"]
