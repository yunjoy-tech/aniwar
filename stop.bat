
for %%i in (daprd.exe) do (
        taskkill /F /im %%~nxi
)

:: taskkill /F /im placement.exe

:: taskkill /F /im consul.exe

:: taskkill /F /im filebeat.exe
:: taskkill /F /im filebeat-prom.exe
taskkill /F /im filebeat-datalog.exe
taskkill /F /im promtail.exe

exit
