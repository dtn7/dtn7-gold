# SPDX-FileCopyrightText: 2021 Jonas Höchst
#
# SPDX-License-Identifier: GPL-3.0-or-later

FROM golang:latest AS dtn7-builder

COPY . /dtn7-go
WORKDIR /dtn7-go
RUN go build ./cmd/dtn-tool
RUN go build ./cmd/dtnd

FROM maciresearch/core_worker:7.4.0-1

COPY --from=dtn7-builder /dtn7-go/dtn-tool /usr/local/bin/
COPY --from=dtn7-builder /dtn7-go/dtnd /usr/local/bin/

COPY ./contrib/coreemu/dotcore/ /root/.core/

RUN echo "custom_services_dir = /root/.core/myservices" | tee -a /etc/core/core.conf
