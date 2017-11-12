#!/usr/bin/env bash
set -e

. ./test_lib.sh

crosscompile
restartdocker
startnodes nodea nodeb nodepub
sleep 3

log "Sending messages"
sendmsg nodea $message_c1_1
sendmsg nodeb $message_c2_1

log "Testing output"
wait_grep nodepub "DSDV nodea: 172.16.0.3:10000"
wait_grep nodepub "DSDV nodeb: 172.16.0.4:10000"

stopdocker
