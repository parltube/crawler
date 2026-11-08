#/bin/sh
# i will just run makefile to generate and extract instead of writing cli by hand and copy pasting them

PY    ?= python
LABEL ?= run
IDS   ?=
PAUSE ?= 15
ARGS  ?=

FLAGS = --label $(LABEL) $(if $(QUICK),--quick) $(if $(IDS),--ids $(IDS)) $(ARGS)

PROBES = watch browse player comments

.PHONY: help install sanity integrity all streamcut $(PROBES)

help:
	@echo "targets: install sanity $(PROBES) integrity all streamcut"
	@echo "vars:    LABEL=name QUICK=1 IDS=ids.txt PAUSE=15 ARGS='--conc 16'"
	@echo "e.g.:    make watch LABEL=isp-tun QUICK=1"
	@echo "         make streamcut ARGS='-n 10 -need player -targets watch:dQw4w9WgXcQ,channel:@MrBeast'"

streamcut:
	go run ./streamcut $(ARGS)

install:
	$(PY) -m pip install curl_cffi

sanity:
	$(PY) utils.py $(ARGS)

$(PROBES):
	$(PY) $@.py $(FLAGS)

integrity:
	$(PY) watch.py --integrity $(FLAGS)

all:
	@for p in $(PROBES); do \
		$(PY) $$p.py $(FLAGS); \
		echo "### пауза $(PAUSE) мин ###"; sleep $$(($(PAUSE) * 60)); \
	done
	$(PY) watch.py --integrity $(FLAGS)
