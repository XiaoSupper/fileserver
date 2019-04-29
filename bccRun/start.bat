cls
@echo off
color 0a
:MENU
ECHO 
ECHO 1 
ECHO 2 
ECHO 3 
ECHO 4 
ECHO 5 
echo 
set /p ID=
if "%id%"=="1" goto cmd1
if "%id%"=="2" goto cmd2
if "%id%"=="3" goto cmd3
if "%id%"=="4" goto cmd4
exit

:cmd1
time /T
\.exe 
time /T
goto MENU

:cmd2
\.exe -http 8001 \idx_test
GOTO MENU

:cmd3 
\.exe -as config.txt \info\as.txt idx
GOTO MENU


:cmd4
\.exe 
GOTO MENU


