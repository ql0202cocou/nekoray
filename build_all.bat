@echo off
setlocal

:: Setup VS Build Tools environment
call "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\VC\Auxiliary\Build\vcvarsall.bat" amd64
if errorlevel 1 (
    echo Failed to setup VS environment
    exit /b 1
)

set CMAKE=C:\Program Files\Microsoft Visual Studio\18\Community\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin\cmake.exe
set NINJA=C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\Common7\IDE\CommonExtensions\Microsoft\CMake\Ninja\ninja.exe
set QT_PATH=D:\WorkSpaces\Qt6.5.0\Qt6.5.0-Windows-x86_64-VS2022-17.5.5
set DEPS_PREFIX=D:\WorkSpaces\nekoray\libs\deps\built

:: Create deps directory
mkdir "%DEPS_PREFIX%" 2>nul

:: ===== Build ZXing =====
echo === Building ZXing ===
cd /d D:\WorkSpaces\nekoray\libs\deps
if not exist zxing-cpp-2.0.0 (
    curl -x http://127.0.0.1:7890 -L -o dl.zip https://github.com/nu-book/zxing-cpp/archive/refs/tags/v2.0.0.zip
    powershell -Command "Expand-Archive -Path 'dl.zip' -DestinationPath '.' -Force"
)
cd zxing-cpp-2.0.0
mkdir build 2>nul
cd build
"%CMAKE%" .. -GNinja -DBUILD_SHARED_LIBS=OFF -DCMAKE_BUILD_TYPE=Release -DBUILD_EXAMPLES=OFF -DBUILD_BLACKBOX_TESTS=OFF -DCMAKE_INSTALL_PREFIX="%DEPS_PREFIX%"
if errorlevel 1 goto :error
"%NINJA%" && "%NINJA%" install
if errorlevel 1 goto :error
cd /d D:\WorkSpaces\nekoray\libs\deps

:: ===== Build yaml-cpp =====
echo === Building yaml-cpp ===
if not exist yaml-cpp-yaml-cpp-0.7.0 (
    curl -x http://127.0.0.1:7890 -L -o dl.zip https://github.com/jbeder/yaml-cpp/archive/refs/tags/yaml-cpp-0.7.0.zip
    powershell -Command "Expand-Archive -Path 'dl.zip' -DestinationPath '.' -Force"
)
cd yaml-cpp-yaml-cpp-0.7.0
mkdir build 2>nul
cd build
"%CMAKE%" .. -GNinja -DBUILD_SHARED_LIBS=OFF -DBUILD_TESTING=OFF -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX="%DEPS_PREFIX%" -DCMAKE_POLICY_VERSION_MINIMUM=3.5
if errorlevel 1 goto :error
"%NINJA%" && "%NINJA%" install
if errorlevel 1 goto :error
cd /d D:\WorkSpaces\nekoray\libs\deps

:: ===== Build protobuf =====
echo === Building protobuf ===
if not exist protobuf (
    git -c http.proxy=http://127.0.0.1:7890 -c https.proxy=http://127.0.0.1:7890 clone --recurse-submodules -b v21.4 --depth 1 --shallow-submodules https://github.com/protocolbuffers/protobuf
)
cd protobuf
mkdir build 2>nul
cd build
"%CMAKE%" .. -GNinja -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF -Dprotobuf_MSVC_STATIC_RUNTIME=OFF -Dprotobuf_BUILD_TESTS=OFF -DCMAKE_INSTALL_PREFIX="%DEPS_PREFIX%" -DCMAKE_POLICY_VERSION_MINIMUM=3.5
if errorlevel 1 goto :error
"%NINJA%" && "%NINJA%" install
if errorlevel 1 goto :error
cd /d D:\WorkSpaces\nekoray

:: ===== Build nekoray =====
echo === Building nekoray ===
mkdir build 2>nul
cd build
"%CMAKE%" .. -GNinja -DCMAKE_BUILD_TYPE=Release -DCMAKE_PREFIX_PATH="%QT_PATH%" -DNKR_LIBS="%DEPS_PREFIX%" -DQT_VERSION_MAJOR=6 -DCMAKE_POLICY_VERSION_MINIMUM=3.5
if errorlevel 1 goto :error
"%NINJA%"
if errorlevel 1 goto :error

echo.
echo === BUILD SUCCESS ===
echo Output: D:\WorkSpaces\nekoray\build\nekobox.exe
exit /b 0

:error
echo.
echo === BUILD FAILED ===
exit /b 1
