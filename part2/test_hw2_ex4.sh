#!/usr/bin/env bash
set -e

. ./test_lib.sh

crosscompile
restartdocker
startnodes nodea nodeb nodepub

log "Sending messages"
sendmsg nodea $message_c1_1
sendmsg nodeb $message_c2_1
sleep 2
sendpriv nodea nodeb $message_c2_2

log "Testing output"
test_grep nodea "CLIENT"
test_grep nodeb "CLIENT"
wait_grep nodepub "MONGERING ROUTE to"
test_ngrep nodepub "MONGERING TEXT to"
wait_grep nodea "DSDV nodeb"
wait_grep nodeb "DSDV nodea"
test_ngrep nodeb "$message_c1_1"
wait_grep nodepub "Not forwarding private message"
test_ngrep nodeb "$message_c2_2"

stopdocker
