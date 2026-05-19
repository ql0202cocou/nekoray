在 macOS 下编译 Nekoray

### git clone 源码

```
git clone https://github.com/MatsuriDayo/nekoray.git --recursive
```

### 安装依赖

安装 Xcode Command Line Tools、CMake、Ninja 和 Qt。

```shell
xcode-select --install
brew install cmake ninja qt
```

如果使用官方 Qt SDK，请把下面命令中的 `CMAKE_PREFIX_PATH` 替换为你的 Qt 安装目录。

### C++ 部分编译

```shell
cmake -S . -B build/macos -GNinja \
  -DCMAKE_BUILD_TYPE=Release \
  -DQT_VERSION_MAJOR=6 \
  -DCMAKE_PREFIX_PATH="$(brew --prefix qt)"
cmake --build build/macos
```

编译完成后得到 `build/macos/nekobox.app`。请双击 `.app` 启动，不要双击 `.app/Contents/MacOS/nekobox` 里的裸可执行文件，否则 Finder 会通过终端启动它。

### 打包 Qt 运行库

```shell
"$(brew --prefix qt)/bin/macdeployqt" build/macos/nekobox.app
```

### Go 部分编译

请看 [Build_Core.md](./Build_Core.md)
