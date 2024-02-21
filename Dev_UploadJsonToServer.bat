@echo off

set LOCAL_DIR=.\Realm\mymodule\JsonData\*
set GCS_BUCKET=gs://herofishing_gamejson_dev3

gsutil -m cp -r %LOCAL_DIR% %GCS_BUCKET%

pause
