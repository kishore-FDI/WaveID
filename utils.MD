$env:PATH="C:\msys64\ucrt64\bin;$env:PATH"

go env -w CGO_ENABLED=1
go env -w CC=x86_64-w64-mingw32-gcc
