# nekoray 修复方案

基于代码审查发现的 22 个 bug，按严重程度排列修复顺序。

---

## 一、严重 (Critical)

### 1.1 gRPC 死锁 — `rpc/gRPC.cpp:172-183`

**问题：** 线程A对 `QMutex` 上锁，然后另一个线程解锁，线程A再次尝试加锁。Qt5 非递归锁直接死锁，跨线程解锁也是 Qt 明确禁止的未定义行为。

**修复：** 用 `QSemaphore` 替代 `QMutex`，或使用 `QWaitCondition`。

```cpp
// === 修复前 (rpc/gRPC.cpp:172-183) ===
QMutex lock;
lock.lock();

runOnUiThread(
    [&] {
        err = call(methodName, serviceName, requestArray, responseArray, timeout_ms);
        lock.unlock();
    },
    nm);

lock.lock();
lock.unlock();

// === 修复后 ===
QSemaphore sem(0);  // 初始计数为0

runOnUiThread(
    [&] {
        err = call(methodName, serviceName, requestArray, responseArray, timeout_ms);
        sem.release();  // 释放信号量
    },
    nm);

sem.acquire();  // 等待工作线程完成
```

---

### 1.2 Go panic: 空 URL 导致 nil request — `go/grpc_server/fulltest.go:131`

**问题：** `http.NewRequestWithContext` 返回的错误被丢弃，当 URL 为空或格式错误时 `req` 为 nil，`httpClient.Do(nil)` 导致整个 gRPC 服务器崩溃。

**修复：**

```go
// === 修复前 ===
go func() {
    req, _ := http.NewRequestWithContext(ctx, "GET", in.FullSpeedUrl, nil)
    resp, err := httpClient.Do(req)

// === 修复后 ===
go func() {
    req, err := http.NewRequestWithContext(ctx, "GET", in.FullSpeedUrl, nil)
    if err != nil {
        close(bodyChan)
        result <- "Error"
        close(result)
        return
    }
    resp, err := httpClient.Do(req)
```

---

### 1.3 Go panic: Outbound().Default() 返回 nil — `go/cmd/nekobox_core/neko_boxapi.go:20-21, 29-35`

**问题：** `b.Outbound().Default()` 在没有配置默认出站时返回 nil，直接调用 `DialContext` 导致 nil panic。

**修复：**

```go
// === 修复前 ===
func nekoDialContext(ctx context.Context, b *box.Box, network, addr string) (net.Conn, error) {
    if b == nil {
        return nil, fmt.Errorf("box instance is nil")
    }
    outbound := b.Outbound().Default()
    return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))
}

// === 修复后 ===
func nekoDialContext(ctx context.Context, b *box.Box, network, addr string) (net.Conn, error) {
    if b == nil {
        return nil, fmt.Errorf("box instance is nil")
    }
    outbound := b.Outbound().Default()
    if outbound == nil {
        return nil, fmt.Errorf("no default outbound configured")
    }
    return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))
}
```

同样修改 `nekoCreateProxyHttpClient`：

```go
// === 修复前 ===
outbound := b.Outbound().Default()
return &http.Client{
    Transport: &http.Transport{
        ...
        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))

// === 修复后 ===
outbound := b.Outbound().Default()
if outbound == nil {
    return &http.Client{}  // 返回无代理的 HTTP 客户端
}
return &http.Client{
    Transport: &http.Transport{
        ...
        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))
```

---

### 1.4 Go panic: Router() 返回 nil — `go/cmd/nekobox_core/grpc_box.go:70`

**问题：** `instance.Router()` 可能返回 nil，直接调用 `AppendTracker` 导致 panic。

**修复：**

```go
// === 修复前 ===
if instance != nil {
    // V2ray Service
    if in.StatsOutbounds != nil {
        statsService = v2rayapi.NewStatsService(option.V2RayStatsServiceOptions{
            Enabled:   true,
            Outbounds: in.StatsOutbounds,
        })
        instance.Router().AppendTracker(statsService)
    }
}

// === 修复后 ===
if instance != nil {
    // V2ray Service
    if in.StatsOutbounds != nil {
        statsService = v2rayapi.NewStatsService(option.V2RayStatsServiceOptions{
            Enabled:   true,
            Outbounds: in.StatsOutbounds,
        })
        if router := instance.Router(); router != nil {
            router.AppendTracker(statsService)
        }
    }
}
```

---

## 二、高危 (High)

### 2.1 QTimer 内存泄漏 — `ui/mainwindow.cpp:425-431`

**问题：** 局部变量 `t` 被重用，第一个 QTimer 指针被覆盖导致泄漏。

**修复：**

```cpp
// === 修复前 ===
auto t = new QTimer;
connect(t, &QTimer::timeout, this, [=]() { refresh_status(); });
t->start(2000);

t = new QTimer;
connect(t, &QTimer::timeout, this, [&] { NekoGui_sys::logCounter.fetchAndStoreRelaxed(0); });
t->start(1000);

// === 修复后 ===
auto t1 = new QTimer(this);  // parent 设为 this，自动管理生命周期
connect(t1, &QTimer::timeout, this, [=]() { refresh_status(); });
t1->start(2000);

auto t2 = new QTimer(this);
connect(t2, &QTimer::timeout, this, [&] { NekoGui_sys::logCounter.fetchAndStoreRelaxed(0); });
t2->start(1000);
```

---

### 2.2 CoreProcess 内存泄漏 — `ui/mainwindow.cpp:402`

**问题：** `core_process` 裸指针从未被 `delete`。

**修复：** 在 `stop_core_daemon` 中释放，或将成员变量改为智能指针。

```cpp
// === ui/mainwindow.h:144 ===
// 修复前
NekoGui_sys::CoreProcess *core_process;

// 修复后（使用 QPointer 自动追踪 QObject 生命周期）
QPointer<NekoGui_sys::CoreProcess> core_process;
```

在停止 core 的地方确保清理：

```cpp
// === ui/mainwindow_grpc.cpp (stop_core_daemon 函数末尾添加) ===
if (core_process) {
    core_process->deleteLater();
    core_process = nullptr;
}
```

---

### 2.3 TrafficData::bypass 内存泄漏 — `db/traffic/TrafficLooper.hpp:24`

**问题：** 类中 `new` 分配的裸指针没有对应的 `delete`。

**修复：** 添加析构函数或使用智能指针。

```cpp
// === 修复前 ===
private:
    TrafficData *bypass = new TrafficData("bypass");

// === 修复后 (方案1：添加析构函数) ===
public:
    ~TrafficLooper() {
        delete bypass;
        bypass = nullptr;
    }
private:
    TrafficData *bypass = new TrafficData("bypass");

// === 修复后 (方案2：使用 unique_ptr，推荐) ===
private:
    std::unique_ptr<TrafficData> bypass = std::make_unique<TrafficData>("bypass");
```

如果使用方案2，需要在 `Loop()` 中将 `bypass` 改为 `bypass.get()`。

---

### 2.4 下标 off-by-one 导致配置删除错误 — `sub/GroupUpdater.cpp:589`

**问题：** `deleted_index > 0` 将索引 0（第一个元素）误判为"未找到"。

**修复：**

```cpp
// === 修复前 ===
auto deleted_index = update_del.indexOf(ent);
if (deleted_index > 0) {

// === 修复后 ===
auto deleted_index = update_del.indexOf(ent);
if (deleted_index != -1) {
```

---

### 2.5 无条件 break 导致 header 解析只执行一次 — `sub/GroupUpdater.cpp:396-401`

**问题：** `break` 在 `if` 块外部，导致循环总是只检查第一个 header。

**修复：**

```cpp
// === 修复前 ===
auto headers = tcp_http["headers"];
for (auto header: headers) {
    if (Node2QString(header.first).toLower() == "host") {
        bean->stream->host = Node2QString(header.second[0]);
    }
    break;   // ⚠️ 无条件 break
}

// === 修复后 ===
auto headers = tcp_http["headers"];
for (auto header: headers) {
    if (Node2QString(header.first).toLower() == "host") {
        bean->stream->host = Node2QString(header.second[0]);
        break;  // 找到 Host 后退出
    }
}
```

---

### 2.6 空 reality_sid 数组越界 — `fmt/Bean2CoreObj_box.cpp:63`

**问题：** 空字符串的 `split(",")` 返回空列表，`operator[0]` 是越界访问。

**修复：**

```cpp
// === 修复前 ===
{"short_id", reality_sid.split(",")[0]},

// === 修复后 ===
{"short_id", reality_sid.split(",").value(0, "")},
```

---

### 2.7 QUICBean 静默接受任意 URL — `fmt/Link2Bean.cpp:293`

**问题：** 函数结尾无条件 `return true`，不支持的 scheme 也被接受。

**修复：**

```cpp
// === 修复前 ===
    } else if (QStringList{"hy2", "hysteria2"}.contains(url.scheme())) {
        // ... hy2 解析 ...
    }

    return true;
}

// === 修复后 ===
    } else if (QStringList{"hy2", "hysteria2"}.contains(url.scheme())) {
        // ... hy2 解析 ...
    } else {
        return false;  // 不支持的 scheme
    }

    return true;
}
```

---

### 2.8 Goroutine 泄漏 — `go/grpc_server/fulltest.go:61-83, 130-150`

**问题：** 无缓冲 channel + context 超时竞速，超时后工作 goroutine 永久阻塞在 `result <- ...` 上。

**修复：** 使用容量为 1 的缓冲 channel。

```go
// === UDP 测试 (line 59) ===
// 修复前
result := make(chan string)
// 修复后
result := make(chan string, 1)

// === 速度测试 (line 126) ===
// 修复前
result := make(chan string)
// 修复后
result := make(chan string, 1)
```

---

### 2.9 数据竞态：update_download_url — `go/grpc_server/update.go:17`

**问题：** 包级变量无同步保护，并发 Check/Download 调用产生数据竞态。

**修复：** 使用 `sync.Mutex` 保护或消除共享变量。

```go
// === 修复前 ===
var update_download_url string

// === 修复后（方案1：添加锁） ===
var (
    update_download_url string
    update_download_mu  sync.Mutex
)

// 在 UpdateAction_Check 中：
update_download_mu.Lock()
update_download_url = asset.BrowserDownloadUrl
update_download_mu.Unlock()

// 在 Download 中：
update_download_mu.Lock()
url := update_download_url
update_download_mu.Unlock()
if url == "" {
    ret.Error = "?"
    return ret, nil
}

// === 修复后（方案2：内联到 Update，根本消除共享变量） ===
// 将 Check 中获取的 URL 直接存入 ret.DownloadUrl，
// Download 时从 in.DownloadUrl 读取，无需包级变量
```

方案2更彻底 —— 让客户端在 Download 请求中传入 `download_url` 字段（如果 proto 中已有该字段），或让服务端在 Check 时将结果持久化到临时文件。

---

### 2.10 TrafficLooper 潜在的 use-after-free — `db/traffic/TrafficLooper.cpp:119-131`

**问题：** lambda 在 MainWindow 销毁后执行，`GetMainWindow()` 返回悬垂指针。

**修复：** 加入空指针检查。

```cpp
// === 修复前 ===
runOnUiThread([=] {
    auto m = GetMainWindow();
    if (proxy != nullptr) {
        m->refresh_status(...);

// === 修复后 ===
runOnUiThread([=] {
    auto m = GetMainWindow();
    if (m == nullptr) return;
    if (proxy != nullptr) {
        m->refresh_status(...);
```

同样修改第92行的 `refresh_status("STOP")` 调用。

---

### 2.11 createBox 错误被遮蔽 — `go/cmd/nekobox_core/grpc_box.go:145-151`

**问题：** `:=` 创建了局部 `err`，遮蔽了外层的 `err`，defer 错误处理器看不到此错误。

**修复：**

```go
// === 修复前 ===
} else if in.Mode == gen.TestMode_FullTest {
    i, cancel, err := createBox([]byte(in.Config.CoreConfig))
    if i != nil {
        defer i.Close()
        defer cancel()
    }
    if err != nil {
        return
    }

// === 修复后 ===
} else if in.Mode == gen.TestMode_FullTest {
    var i *box.Box
    var cancel context.CancelFunc
    i, cancel, err = createBox([]byte(in.Config.CoreConfig))
    if i != nil {
        defer i.Close()
        defer cancel()
    }
    if err != nil {
        return
    }
```

---

### 2.12 VMess v2rayN 格式缺少必要字段校验 — `fmt/Link2Bean.cpp:155-177`

**问题：** v2rayN 格式分支只检查了 `objN.isEmpty()`，但后续直接读取 `id`、`add`、`port` 字段，不检查这些字段是否为空。空 uuid/address 的配置会被静默接受。而对比之下，Xray 格式分支（第232行）正确做了 `return !(uuid.isEmpty() || serverAddress.isEmpty())` 校验。

**修复：**

```cpp
// === 修复前 (fmt/Link2Bean.cpp:155-177) ===
uuid = objN["id"].toString();
serverAddress = objN["add"].toString();
serverPort = objN["port"].toVariant().toInt();
// ... OPTIONAL 字段解析 ...
return true;  // 无条件返回 true

// === 修复后 ===
uuid = objN["id"].toString();
serverAddress = objN["add"].toString();
serverPort = objN["port"].toVariant().toInt();
// 添加必要字段校验
if (uuid.isEmpty() || serverAddress.isEmpty() || serverPort == 0) return false;
// ... OPTIONAL 字段解析 ...
return true;
```

---

### 2.13 DNS 解析回调 use-after-free — `fmt/AbstractBean.cpp:55-62`

**问题：** `lookupHost` 的 lambda 用 `[=]` 捕获 `this` 指针。如果用户在执行 DNS 解析期间删除了代理配置，`AbstractBean` 对象被释放，回调中访问 `serverAddress` 和 `GetStreamSettings(this)` 都是悬垂指针解引用。

**修复：**

```cpp
// === 修复前 ===
QHostInfo::lookupHost(serverAddress, QApplication::instance(), [=](const QHostInfo &host) {
    auto addr = host.addresses();
    if (!addr.isEmpty()) {
        auto domain = serverAddress;
        auto stream = GetStreamSettings(this);
        serverAddress = addr.first().toString();
        // ...

// === 修复后 ===
QPointer<AbstractBean> guard(this);
QHostInfo::lookupHost(serverAddress, QApplication::instance(), [guard](const QHostInfo &host) {
    if (!guard) return;  // 对象已被删除，安全退出
    auto addr = host.addresses();
    if (!addr.isEmpty()) {
        auto domain = guard->serverAddress;
        auto stream = GetStreamSettings(guard);
        guard->serverAddress = addr.first().toString();
        // ...
```

需要将后续访问 `this` 的代码改为通过 `guard` 访问。

---

**问题：** 如果 auth token 包含 `\r\n`，可注入任意 HTTP 头。

**修复：**

```cpp
// === 修复前 ===
request.setRawHeader("nekoray_auth", nekoray_auth);

// === 修复后 ===
// 过滤控制字符，防止HTTP头部注入
QByteArray safeAuth = nekoray_auth;
safeAuth.replace('\r', "");
safeAuth.replace('\n', "");
request.setRawHeader("nekoray_auth", safeAuth);
```

---

### 3.2 未检查 QFile::open() 返回值 — `main/NekoGui_Utils.cpp:133-144`

**问题：** 文件打不开时静默返回空数据，无法与"文件为空"区分。

**修复：**

```cpp
// === 修复前 ===
QByteArray ReadFile(const QString &path) {
    QFile file(path);
    file.open(QFile::ReadOnly);
    return file.readAll();
}

// === 修复后 ===
QByteArray ReadFile(const QString &path) {
    QFile file(path);
    if (!file.open(QFile::ReadOnly)) {
        MW_show_log("Failed to open file: " + path + " - " + file.errorString());
        return {};
    }
    return file.readAll();
}

// === 修复前 ===
QString ReadFileText(const QString &path) {
    QFile file(path);
    file.open(QFile::ReadOnly | QFile::Text);
    QTextStream stream(&file);
    return stream.readAll();
}

// === 修复后 ===
QString ReadFileText(const QString &path) {
    QFile file(path);
    if (!file.open(QFile::ReadOnly | QFile::Text)) {
        MW_show_log("Failed to open file: " + path + " - " + file.errorString());
        return {};
    }
    QTextStream stream(&file);
    return stream.readAll();
}
```

---

### 3.3 非 UI 线程调用 QInputDialog — `sub/GroupUpdater.cpp:477`

**问题：** 在非UI线程创建 `QInputDialog` 会导致 Qt 崩溃。

**修复：** 将对话框移到主线程。

```cpp
// === 修复前 ===
auto a = QInputDialog::getItem(nullptr,
       QObject::tr("url detected"),
       QObject::tr("%1\nHow to update?").arg(content),
       items, 0, false, &ok);

// === 修复后 ===
QString a;
bool ok = false;
QMetaObject::invokeMethod(
    QCoreApplication::instance(),
    [&] {
        a = QInputDialog::getItem(nullptr,
               QObject::tr("url detected"),
               QObject::tr("%1\nHow to update?").arg(content),
               items, 0, false, &ok);
    },
    Qt::BlockingQueuedConnection);
```

---

### 3.4 bodyChan close 行为不一致 — `go/grpc_server/fulltest.go:146,161`

**问题：** 错误路径上 `close(bodyChan)` 发送零值，但主 goroutine 在 select 之后无条件读取 bodyChan。错误路径上 bodyChan 已 close 且没有写入，会读取到零值（nil），但这会产生不必要的 GC 活动。

**修复：** 统一 channel 使用逻辑，确保所有路径都一致地写入或关闭。

```go
// === 修复后 ===
go func() {
    req, err := http.NewRequestWithContext(ctx, "GET", in.FullSpeedUrl, nil)
    if err != nil {
        bodyChan <- nil  // 发送 nil 而非 close
        result <- "Error"
        close(result)
        return
    }
    resp, err := httpClient.Do(req)
    if err == nil && resp != nil && resp.Body != nil {
        bodyChan <- resp.Body
        defer resp.Body.Close()
        // ... 剩余逻辑不变 ...
    } else {
        bodyChan <- nil  // 发送 nil
        result <- "Error"
    }
    close(result)
}()

// 主 goroutine 中：
cancel()
if bc, ok := <-bodyChan; ok && bc != nil {
    bc.Close()
}
```

---

### 3.5 Hysteria2 ALPN 设为字符串而非 JSON 数组 — `fmt/Bean2CoreObj_box.cpp:186`

**问题：** sing-box 的 TLS ALPN 字段期望 JSON 数组（如 `["h3"]`），但第186行将其覆盖为纯字符串 `"h3"`，可能导致配置被 sing-box 拒绝。同时无条件覆盖也丢弃了用户在 TUIC 路径（第185行）中配置的 ALPN 值。

**修复：**

```cpp
// === 修复前 ===
if (!alpn.trimmed().isEmpty()) coreTlsObj["alpn"] = QList2QJsonArray(alpn.split(","));
if (proxy_type == proxy_Hysteria2) coreTlsObj["alpn"] = "h3";

// === 修复后 ===
if (proxy_type == proxy_Hysteria2) {
    coreTlsObj["alpn"] = QJsonArray{"h3"};  // 使用 JSON 数组
} else if (!alpn.trimmed().isEmpty()) {
    coreTlsObj["alpn"] = QList2QJsonArray(alpn.split(","));
}
```

---

### 3.6 Flow 字段在 Build 函数中被突变 — `fmt/Bean2CoreObj_box.cpp:157-168`

**问题：** `BuildCoreObjSingBox` 是一个"构建"函数，调用者期望它只读。但它直接修改成员变量 `flow`（执行 `chop(7)` 或赋空字符串）。如果该函数被调用两次，第二次调用时 flow 已经被第一次修改，产生不同结果。

**修复：** 使用局部变量而非修改成员变量。

```cpp
// === 修复前 ===
if (flow.right(7) == "-udp443") {
    flow.chop(7);         // 直接修改成员变量
} else if (flow == "none") {
    flow = "";            // 直接修改成员变量
}
outbound["uuid"] = password.trimmed();
outbound["flow"] = flow;

// === 修复后 ===
QString actualFlow = flow;
if (actualFlow.right(7) == "-udp443") {
    actualFlow.chop(7);
} else if (actualFlow == "none") {
    actualFlow = "";
}
outbound["uuid"] = password.trimmed();
outbound["flow"] = actualFlow;
```

---

### 3.7 WriteTempFile 写入失败后继续执行 — `fmt/Bean2External.cpp:9-20`

**问题：** `WriteTempFile` 宏在 `f.open()` 失败时只设置了 `result.error`，但仍继续执行后续代码。无效的 `TempFile` 路径被用于设置环境变量（`SSL_CERT_FILE=TempFile`）或命令行参数（`-c TempFile`）。调用者不检查 `result.error`，导致静默使用无效文件路径。

**修复：**

```cpp
// === 修复前 ===
#define WriteTempFile(fn, data)                                   \
    QDir dir;                                                     \
    if (!dir.exists("temp")) dir.mkdir("temp");                   \
    QFile f(QStringLiteral("temp/") + fn);                               \
    bool ok = f.open(QIODevice::WriteOnly | QIODevice::Truncate); \
    if (ok) {                                                     \
        f.write(data);                                            \
    } else {                                                      \
        result.error = f.errorString();                           \
    }                                                             \
    f.close();                                                    \
    auto TempFile = QFileInfo(f).absoluteFilePath();

// === 修复后 ===
#define WriteTempFile(fn, data)                                   \
    QDir dir;                                                     \
    if (!dir.exists("temp")) dir.mkdir("temp");                   \
    QFile f(QStringLiteral("temp/") + fn);                               \
    bool ok = f.open(QIODevice::WriteOnly | QIODevice::Truncate); \
    if (ok) {                                                     \
        f.write(data);                                            \
        f.close();                                                \
    } else {                                                      \
        result.error = f.errorString();                           \
        return result;                                            \
    }                                                             \
    auto TempFile = QFileInfo(f).absoluteFilePath();
```

**注意：** 这需要确认所有使用 `WriteTempFile` 的函数返回类型是 `ExternalBuildResult`，并且调用者可以处理 `return result`。如果不能从宏中 return，备选方案是在调用点检查 `result.error`。

---

### 3.8 gRPC 响应缺少 status header 被当作成功 — `rpc/gRPC.cpp:96-98`

**问题：** `rawHeader()` 返回空 `QByteArray`，空字符串的 `toInt()` 返回 0。当服务端因连接中断等原因未返回 `grpc-status` 头时，代码将响应当作成功处理。

**修复：**

```cpp
// === 修复前 ===
auto errCode = networkReply->rawHeader(GrpcStatusHeader).toInt();
if (errCode != 0) {

// === 修复后 ===
auto errCodeBytes = networkReply->rawHeader(GrpcStatusHeader);
if (errCodeBytes.isEmpty()) {
    statusCode = QNetworkReply::NetworkError::ProtocolUnknownError;
    return {};
}
auto errCode = errCodeBytes.toInt();
if (errCode != 0) {
```

---

### 3.9 timeout_ms 为 0 时无超时保护 — `rpc/gRPC.cpp:113-119`

**问题：** `Start` 和 `Stop` 调用 `Call` 时不传 timeout_ms（默认为0），此时不创建 abortTimer。如果网络或服务端无响应，`loop.exec()` 永久阻塞，导致 UI 卡死。

**修复：**

```cpp
// === 修复前 ===
QTimer *abortTimer = nullptr;
if (timeout_ms > 0) {
    abortTimer = new QTimer;
    abortTimer->setSingleShot(true);
    abortTimer->setInterval(timeout_ms);
    QObject::connect(abortTimer, &QTimer::timeout, networkReply, &QNetworkReply::abort);
    abortTimer->start();
}

// === 修复后 ===
// 确保始终有超时保护，默认30秒
int actualTimeout = timeout_ms > 0 ? timeout_ms : 30000;
auto abortTimer = new QTimer;
abortTimer->setSingleShot(true);
abortTimer->setInterval(actualTimeout);
QObject::connect(abortTimer, &QTimer::timeout, networkReply, &QNetworkReply::abort);
abortTimer->start();
```

---

### 3.10 Update 下载并发文件冲突 — `go/grpc_server/update.go:108`

**问题：** 第二个并发的 `Download` RPC 会 `O_TRUNC` 正在被第一个调用写入的文件，导致数据损坏。

**修复：** 添加互斥锁保护下载流程。

```go
// === 在 update.go 顶部添加 ===
var downloadMu sync.Mutex

// === 在 UpdateAction_Download 分支开头添加 ===
downloadMu.Lock()
defer downloadMu.Unlock()
```

同时修复 `f.Sync()` 的错误处理：

```go
// === 修复前 ===
f.Sync()

// === 修复后 ===
if err := f.Sync(); err != nil {
    ret.Error = err.Error()
    return ret, nil
}
```

---

### 3.11 QUIC URL 端口校验过严 + Hysteria2 缺少默认端口 — `fmt/Link2Bean.cpp:257,279`

**问题：** 第257行 `url.port() == -1` 拒绝所有无端口 URL，但第265行 TUIC 路径有 `url.port(443)` 设置默认端口（变成死代码）。Hysteria2 路径（第279行）用 `url.port()` 无默认值。

**修复：**

```cpp
// === 修复前 ===
if (url.host().isEmpty() || url.port() == -1) return false;

// ... TUIC ...
serverPort = url.port(443);  // 默认端口变为死代码

// ... Hysteria2 ...
serverPort = url.port();     // 无默认端口

// === 修复后 ===
if (url.host().isEmpty()) return false;

// ... TUIC ...
serverPort = url.port(443);  // 现在生效了

// ... Hysteria2 ...
serverPort = url.port(443);  // 添加默认端口
```

## 四、低危 (Low)

### 4.1 整数窄化 + 未对齐访问 — `rpc/gRPC.cpp:80`

**问题：** `reinterpret_cast<int*>` 在非4字节对齐的 `char*` 上是未定义行为（ARM 平台崩溃）。

**修复：**

```cpp
// === 修复前 ===
QByteArray msg(GrpcMessageSizeHeaderSize, '\0');
*reinterpret_cast<int *>(msg.data() + 1) = qToBigEndian((int) args.size());

// === 修复后 ===
QByteArray msg(GrpcMessageSizeHeaderSize, '\0');
union {
    char bytes[4];
    int32_t value;
} length;
length.value = qToBigEndian(static_cast<int32_t>(args.size()));
memcpy(msg.data() + 1, length.bytes, 4);
```

---

### 4.2 原子保存的 TOCTOU — `main/NekoGui.cpp:202-210`

**问题：** `QFile::remove` 失败后 `rename` 可能失败，产生孤立的 `.tmp` 文件。

**修复：**

```cpp
// === 修复前 ===
if (file.open(QIODevice::WriteOnly | QIODevice::Truncate)) {
    file.write(save_content);
    file.close();
    QFile::remove(fn);
    QFile::rename(fn + ".tmp", fn);
}

// === 修复后 ===
if (file.open(QIODevice::WriteOnly | QIODevice::Truncate)) {
    file.write(save_content);
    file.close();
    // rename 在大多数 POSIX 系统上原子地替换目标文件
    // 如果失败，保留 .tmp 但记录警告
    if (!QFile::rename(fn + ".tmp", fn)) {
        MW_show_log("Warning: Failed to save config: " + file.errorString());
    }
}
```

---

### 4.3 重复赋值 — `sub/GroupUpdater.cpp:344-345`

**问题：** 拷贝粘贴错误，相同代码执行两次。

**修复：** 删除第345行。

```cpp
// === 修复前 ===
bean->stream->utlsFingerprint = Node2QString(proxy["client-fingerprint"]);
bean->stream->utlsFingerprint = Node2QString(proxy["client-fingerprint"]);

// === 修复后 ===
bean->stream->utlsFingerprint = Node2QString(proxy["client-fingerprint"]);
```

---

### 4.4 信号处理器空指针风险 — `main/main.cpp:23`

**问题：** `GetMainWindow()` 可能在 mainwindow 初始化之前被信号触发。

**修复：**

```cpp
// === 修复前 ===
void signal_handler(int signum) {
    if (qApp) {
        GetMainWindow()->on_commitDataRequest();
        qApp->exit();
    }
}

// === 修复后 ===
void signal_handler(int signum) {
    if (qApp) {
        if (auto *mw = GetMainWindow()) {
            mw->on_commitDataRequest();
        }
        qApp->exit();
    }
}
```

---

### 4.5 SerializeToString 返回值未检查 — `rpc/gRPC.cpp:167`

**问题：** protobuf 的 `SerializeToString` 可能返回 false（序列化失败），但返回值被忽略，可能发送损坏的消息。

**修复：**

```cpp
// === 修复前 ===
std::string reqStr;
req.SerializeToString(&reqStr);

// === 修复后 ===
std::string reqStr;
if (!req.SerializeToString(&reqStr)) {
    return QNetworkReply::NetworkError::ProtocolUnknownError;
}
```

---

### 4.6 临时文件永不清理 — `fmt/Bean2External.cpp:9-20`

**问题：** 每次外部核心启动时 `WriteTempFile` 都在 `temp/` 目录下创建新文件，但从不删除。

**修复：** 在应用启动时或核心停止后清理过期临时文件。

```cpp
// 在 CoreProcess::Start() 或适当的初始化位置添加：
void CleanupTempFiles() {
    QDir tempDir("temp");
    if (tempDir.exists()) {
        const auto files = tempDir.entryList(QDir::Files);
        for (const auto &f : files) {
            tempDir.remove(f);
        }
    }
}
```

---

### 4.7 Socks4/Socks4a 被混淆 — `fmt/Link2Bean.cpp:20`

**问题：** `startsWith("socks4")` 同时匹配 `socks4://` 和 `socks4a://`，但 Socks4a 支持远程 DNS 解析，Socks4 不支持。两者被当作同一类型处理。

**修复：**

```cpp
// === 修复前 ===
if (link.startsWith("socks4")) socks_http_type = type_Socks4;

// === 修复后 ===
if (link.startsWith("socks4a://")) socks_http_type = type_Socks4;  // 暂用 Socks4
else if (link.startsWith("socks4://")) socks_http_type = type_Socks4;
```

---

### 4.8 多个独立 if 未用 else if — `sub/GroupUpdater.cpp:74-138`

**问题：** 所有协议类型检查都使用独立 `if`。如果将来新增协议前缀是已有前缀的子串，会导致同一输入被多个分支重复处理。

**修复：** 将所有独立 `if` 改为 `else if` 链。

```cpp
// === 修复前 ===
if (str.startsWith("socks5://") ...) { ... }
if (str.startsWith("http://") ...) { ... }
if (str.startsWith("ss://")) { ... }
// ...

// === 修复后 ===
if (str.startsWith("socks5://") ...) { ... }
else if (str.startsWith("http://") ...) { ... }
else if (str.startsWith("ss://")) { ... }
// ...
```

---

### 4.9 Go 端被忽略的错误 — 多处

**问题：** 多个位置忽略了函数返回的 error。

**修复：**

```go
// fulltest.go:67 - SetReadDeadline 错误
if err := pc.SetReadDeadline(time.Now().Add(time.Second * 3)); err != nil {
    log.Println("UDP SetReadDeadline error:", err)
}

// fulltest.go:68 - hex.DecodeString 错误（硬编码字符串安全，但应加入 init 检查）
var dnsPacket []byte
func init() {
    var err error
    dnsPacket, err = hex.DecodeString("0000010000010000000000000377777706676f6f676c6503636f6d0000010001")
    if err != nil {
        panic("invalid hardcoded DNS packet: " + err.Error())
    }
}

// fulltest.go:110 - io.ReadAll 错误
b, err := io.ReadAll(resp.Body)
if err != nil {
    out_ip = "Error"
} else {
    out_ip = getBetweenStr(string(b), "ip=", "\n")
}
```

---

### 4.10 scanner 错误未检查 — `go/grpc_server/grpc.go:77-79`

**问题：** `bufio.Scanner` 的 `s.Err()` 从未被调用，I/O 错误和 EOF 无法区分。

**修复：**

```go
// === 修复前 ===
if s.Scan() {
    token = strings.TrimSpace(s.Text())
}
// s.Err() 未检查

// === 修复后 ===
if s.Scan() {
    token = strings.TrimSpace(s.Text())
}
if err := s.Err(); err != nil {
    log.Fatalf("failed to read token: %v", err)
}
```

---

### 4.11 潜在的无限 base64 递归 — `sub/GroupUpdater.cpp:39`

**问题：** 如果订阅内容 base64 解码后恰好又是合法 base64，会无限递归（虽然现实中几乎不可能）。

**修复：** 添加递归深度限制。

```cpp
// === 修复前 ===
void RawUpdater::update(const QString &str) {
    if (auto str2 = DecodeB64IfValid(str); !str2.isEmpty()) {
        update(str2);  // 无限制递归
        return;
    }

// === 修复后 ===
void RawUpdater::update(const QString &str, int depth = 0) {
    if (depth < 3) {  // 最多递归3层
        if (auto str2 = DecodeB64IfValid(str); !str2.isEmpty()) {
            update(str2, depth + 1);
            return;
        }
    }
```

---

### 4.12 订阅下载 abortTimer 泄漏 — `main/HTTPRequestHelper.cpp:57-69`

**问题：** `abortTimer` 没有父对象，且 `deleteLater()` 在本地 `QEventLoop` 已停止后调用。当从后台线程调用时（如 `GroupUpdater::AsyncUpdate`），线程无事件循环处理 `DeferredDelete` 事件，导致 timer 泄漏。

**修复：**

```cpp
// === 修复前 ===
auto abortTimer = new QTimer;

// === 修复后 ===
auto abortTimer = new QTimer(_reply);  // parent 设为 _reply，自动管理生命周期
```

---

### 4.13 安全字段用子串替换代替精确匹配 — `fmt/Link2Bean.cpp:64-67`

**问题：** `.replace("reality", "tls")` 和 `.replace("none", "")` 是子串替换，如果未来安全字段值包含这些字符串作为子串会被误修改。

**修复：**

```cpp
// === 修复前 ===
stream->security = GetQueryValue(query, "security", "tls").replace("reality", "tls").replace("none", "");

// === 修复后 ===
auto security = GetQueryValue(query, "security", "tls");
if (security == "reality") security = "tls";
if (security == "none") security = "";
stream->security = security;
```

---

### 4.14 死代码 — `fmt/Link2Bean.cpp:236`

**问题：** VMessBean::TryParseLink 中 `return false` 在第236行，但两个分支都已无条件 return，不可达。

**修复：** 直接删除 `return false;` 行。

---

### 4.15 魔数 -114514 — `rpc/gRPC.cpp:191`

**问题：** 使用魔数 `-114514` 作为 protobuf 解析失败的 NetworkError，不是有效枚举值。

**修复：**

```cpp
// === 修复前 ===
return QNetworkReply::NetworkError(-114514);

// === 修复后 ===
return QNetworkReply::NetworkError::ProtocolFailure;
```

---

### 4.16 Shadowsocks plugin 全量替换 — `fmt/Link2Bean.cpp:134`

**问题：** `replace("simple-obfs;", "obfs-local;")` 替换所有出现，如果插件参数中包含该子串也会被误改。（实际风险极低）

**修复：** 使用正则锚定开头 `QRegularExpression("^simple-obfs;")` 替代，或仅在 `startsWith` 时替换首段。

---

### 4.17 purgeHeader 用赋值代替 delete — `go/grpc_server/auth/auth.go:53`

**问题：** `mdCopy[header] = nil` 设置键为 nil 切片而非删除键。功能上等价但不符合 Go 惯用法。

**修复：**

```go
// === 修复前 ===
mdCopy[header] = nil

// === 修复后 ===
delete(mdCopy, header)
```

---

### 4.18 UDP/速度测试未使用 gRPC context — `go/grpc_server/fulltest.go:58,125`

**问题：** 使用 `context.Background()` 而非 gRPC 传入的 `ctx`。客户端断开连接后，UDP 延迟测试和速度测试仍继续运行直到自己的超时到期，浪费资源。

**修复：**

```go
// === 修复前 ===
ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)

// === 修复后 ===
ctx, cancel := context.WithTimeout(ctx, time.Second*3)  // 使用 gRPC ctx
```

同样修改第125行的速度测试 context。

---

## 修复优先级建议

| 优先级 | Bug编号 | 描述 | 影响 |
|--------|---------|------|------|
| P0 | 1.1 | gRPC 死锁 | 随机卡死应用 |
| P0 | 1.2 | Go nil request panic | 服务器崩溃 |
| P0 | 1.3 | Go nil outbound panic | 服务器崩溃 |
| P0 | 1.4 | Go nil router panic | 服务器崩溃 |
| P1 | 2.1~2.3 | 内存泄漏(3个) | 长期运行内存增长 |
| P1 | 2.4 | off-by-one | 配置删除逻辑错误 |
| P1 | 2.5 | break 位置错误 | header 解析遗漏 |
| P1 | 2.6 | 数组越界 | debug 构建崩溃 |
| P1 | 2.8 | goroutine 泄漏 | Go 侧内存增长 |
| P1 | 2.9 | 数据竞态 | 更新下载错乱 |
| P1 | 2.11 | 错误遮蔽 | 测试失败无提示 |
| P1 | 2.12 | VMess v2rayN 无校验 | 无效配置静默接受 |
| P1 | 2.13 | DNS回调 use-after-free | 删配置时潜在崩溃 |
| P2 | 2.7 | 任意 URL 接受 | 无效配置被接受 |
| P2 | 2.10 | use-after-free | 关闭时潜在崩溃 |
| P2 | 3.1 | HTTP 头部注入 | 安全 |
| P2 | 3.2 | QFile::open 未检查 | VPN 配置静默失败 |
| P2 | 3.3 | 非 UI 线程对话框 | 崩溃 |
| P2 | 3.4 | bodyChan 行为不一致 | Go 侧阻塞 |
| P2 | 3.5 | Hysteria2 ALPN 格式 | sing-box 可能拒绝 |
| P2 | 3.6 | Flow 字段突变 | 双重调用异常 |
| P2 | 3.7 | WriteTempFile 失败继续 | 无效文件路径传播 |
| P2 | 3.8 | gRPC status 头缺失当成功 | 错误被掩盖 |
| P2 | 3.9 | timeout_ms=0 无超时 | UI 卡死 |
| P2 | 3.10 | 下载并发文件冲突 | 更新文件损坏 |
| P2 | 3.11 | QUIC 端口校验过严 | 无端口 URL 被拒 |
| P3 | 4.1~4.18 | 低危问题(18个) | 边缘情况正确性 |

## 总结

| 严重程度 | 数量 | 涉及模块 |
|---------|------|---------|
| 严重 (P0) | 4 | gRPC 死锁 + 3个 Go panic |
| 高危 (P1) | 10 | 内存泄漏(3)、逻辑错误(4)、Go并发(2)、use-after-free(1) |
| 中危 (P2) | 13 | fmt(4)、rpc(4)、go(2)、sub(2)、main(1) |
| 低危 (P3) | 18 | 边界情况、代码质量、防御性编程 |

**总计：46 个 bug**

建议分两次 PR：
- **PR1**（P0+P1，约 14 个修复）：核心稳定性——崩溃/死锁/泄漏 + 数据错误
- **PR2**（P2+P3，约 32 个修复）：安全加固 + 边缘情况完整性
