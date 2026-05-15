#include "db/ProxyEntity.hpp"
#include "fmt/includes.h"

namespace NekoGui_fmt {
    void V2rayStreamSettings::BuildStreamSettingsSingBox(QJsonObject *outbound) {
        // https://sing-box.sagernet.org/configuration/shared/v2ray-transport

        if (network != "tcp") {
            QJsonObject transport{{"type", network}};
            if (network == "ws") {
                if (!host.isEmpty()) transport["headers"] = QJsonObject{{"Host", host}};
                // ws path & ed
                auto pathWithoutEd = SubStrBefore(path, "?ed=");
                if (!pathWithoutEd.isEmpty()) transport["path"] = pathWithoutEd;
                if (pathWithoutEd != path) {
                    auto ed = SubStrAfter(path, "?ed=").toInt();
                    if (ed > 0) {
                        transport["max_early_data"] = ed;
                        transport["early_data_header_name"] = "Sec-WebSocket-Protocol";
                    }
                }
                if (ws_early_data_length > 0) {
                    transport["max_early_data"] = ws_early_data_length;
                    transport["early_data_header_name"] = ws_early_data_name;
                }
            } else if (network == "http") {
                if (!path.isEmpty()) transport["path"] = path;
                if (!host.isEmpty()) transport["host"] = QList2QJsonArray(host.split(","));
            } else if (network == "grpc") {
                if (!path.isEmpty()) transport["service_name"] = path;
            } else if (network == "httpupgrade") {
                if (!path.isEmpty()) transport["path"] = path;
                if (!host.isEmpty()) transport["host"] = host;
            }
            outbound->insert("transport", transport);
        } else if (header_type == "http") {
            // TCP + headerType
            QJsonObject transport{
                {"type", "http"},
                {"method", "GET"},
                {"path", path},
                {"headers", QJsonObject{{"Host", QList2QJsonArray(host.split(","))}}},
            };
            outbound->insert("transport", transport);
        }

        // 对应字段 tls
        if (security == "tls") {
            QJsonObject tls{{"enabled", true}};
            if (allow_insecure) tls["insecure"] = true;
            if (!sni.trimmed().isEmpty()) tls["server_name"] = sni;
            if (!certificate.trimmed().isEmpty()) {
                tls["certificate"] = certificate.trimmed();
            }
            if (!alpn.trimmed().isEmpty()) {
                tls["alpn"] = QList2QJsonArray(alpn.split(","));
            }
            QString fp = utlsFingerprint;
            if (!reality_pbk.trimmed().isEmpty()) {
                tls["reality"] = QJsonObject{
                    {"enabled", true},
                    {"public_key", reality_pbk},
                    {"short_id", reality_sid.split(",").value(0, "")},
                };
                if (fp.isEmpty()) fp = "random";
            }
            if (!fp.isEmpty()) {
                tls["utls"] = QJsonObject{
                    {"enabled", true},
                    {"fingerprint", fp},
                };
            }
            outbound->insert("tls", tls);
        }

        if (outbound->value("type").toString() == "vmess" || outbound->value("type").toString() == "vless") {
            outbound->insert("packet_encoding", packet_encoding);
        }
    }

    CoreObjOutboundBuildResult SocksHttpBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound;
        outbound["type"] = socks_http_type == type_HTTP ? "http" : "socks";
        if (socks_http_type == type_Socks4) outbound["version"] = "4";
        outbound["server"] = serverAddress;
        outbound["server_port"] = serverPort;

        if (!username.isEmpty() && !password.isEmpty()) {
            outbound["username"] = username;
            outbound["password"] = password;
        }

        stream->BuildStreamSettingsSingBox(&outbound);
        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult ShadowSocksBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound{{"type", "shadowsocks"}};
        outbound["server"] = serverAddress;
        outbound["server_port"] = serverPort;
        outbound["method"] = method;
        outbound["password"] = password;

        if (uot != 0) {
            QJsonObject udp_over_tcp{
                {"enabled", true},
                {"version", uot},
            };
            outbound["udp_over_tcp"] = udp_over_tcp;
        } else {
            outbound["udp_over_tcp"] = false;
        }

        if (!plugin.trimmed().isEmpty()) {
            outbound["plugin"] = SubStrBefore(plugin, ";");
            outbound["plugin_opts"] = SubStrAfter(plugin, ";");
        }

        stream->BuildStreamSettingsSingBox(&outbound);
        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult VMessBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound{
            {"type", "vmess"},
            {"server", serverAddress},
            {"server_port", serverPort},
            {"uuid", uuid.trimmed()},
            {"alter_id", aid},
            {"security", security},
        };

        stream->BuildStreamSettingsSingBox(&outbound);
        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult TrojanVLESSBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound{
            {"type", proxy_type == proxy_VLESS ? "vless" : "trojan"},
            {"server", serverAddress},
            {"server_port", serverPort},
        };

        QJsonObject settings;
        if (proxy_type == proxy_VLESS) {
            QString actualFlow = flow;
            if (actualFlow.right(7) == "-udp443") {
                actualFlow.chop(7);
            } else if (actualFlow == "none") {
                actualFlow = "";
            }
            outbound["uuid"] = password.trimmed();
            outbound["flow"] = actualFlow;
        } else {
            outbound["password"] = password;
        }

        stream->BuildStreamSettingsSingBox(&outbound);
        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult QUICBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject coreTlsObj{
            {"enabled", true},
            {"disable_sni", disableSni},
            {"insecure", allowInsecure},
            {"certificate", caText.trimmed()},
            {"server_name", sni},
        };
        if (proxy_type == proxy_Hysteria2) {
            coreTlsObj["alpn"] = QJsonArray{"h3"};
        } else if (!alpn.trimmed().isEmpty()) {
            coreTlsObj["alpn"] = QList2QJsonArray(alpn.split(","));
        }

        QJsonObject outbound{
            {"server", serverAddress},
            {"server_port", serverPort},
            {"tls", coreTlsObj},
        };

        if (proxy_type == proxy_Hysteria2) {
            outbound["type"] = "hysteria2";
            outbound["password"] = password;
            outbound["up_mbps"] = uploadMbps;
            outbound["down_mbps"] = downloadMbps;

            if (!hopPort.trimmed().isEmpty()) {
                outbound["hop_ports"] = hopPort;
                outbound["hop_interval"] = hopInterval;
            }
            if (!obfsPassword.isEmpty()) {
                outbound["obfs"] = QJsonObject{
                    {"type", "salamander"},
                    {"password", obfsPassword},
                };
            }
        } else if (proxy_type == proxy_TUIC) {
            outbound["type"] = "tuic";
            outbound["uuid"] = uuid;
            outbound["password"] = password;
            outbound["congestion_control"] = congestionControl;
            if (uos) {
                outbound["udp_over_stream"] = true;
            } else {
                outbound["udp_relay_mode"] = udpRelayMode;
            }
            outbound["zero_rtt_handshake"] = zeroRttHandshake;
            if (!heartbeat.trimmed().isEmpty()) outbound["heartbeat"] = heartbeat;
        }

        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult CustomBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        if (core == "internal") {
            result.outbound = QString2QJsonObject(config_simple);
        }

        return result;
    }

    CoreObjOutboundBuildResult AnyTLSBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound{
            {"type", "anytls"},
            {"server", serverAddress},
            {"server_port", serverPort},
            {"password", password},
        };

        stream->BuildStreamSettingsSingBox(&outbound);
        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult SSHBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound{
            {"type", "ssh"},
            {"server", serverAddress},
            {"server_port", serverPort},
        };

        if (!user.isEmpty()) outbound["user"] = user;
        if (!password.isEmpty()) outbound["password"] = password;
        if (!private_key.isEmpty()) outbound["private_key"] = QList2QJsonArray(private_key);
        if (!private_key_path.isEmpty()) outbound["private_key_path"] = private_key_path;
        if (!private_key_passphrase.isEmpty()) outbound["private_key_passphrase"] = private_key_passphrase;
        if (!host_key.isEmpty()) outbound["host_key"] = QList2QJsonArray(host_key);
        if (!host_key_algorithms.isEmpty()) outbound["host_key_algorithms"] = QList2QJsonArray(host_key_algorithms);
        if (!client_version.isEmpty()) outbound["client_version"] = client_version;

        result.outbound = outbound;
        return result;
    }

    CoreObjOutboundBuildResult TorBean::BuildCoreObjSingBox() {
        CoreObjOutboundBuildResult result;

        QJsonObject outbound{
            {"type", "tor"},
        };

        // C1 fix: Validate executable_path — reject control characters and null bytes
        if (!executable_path.isEmpty()) {
            for (const auto &ch : executable_path) {
                if (ch.unicode() < 0x20 || ch.unicode() == 0x7F) {
                    result.error = "Tor executable_path contains invalid control characters";
                    return result;
                }
            }
            outbound["executable_path"] = executable_path;
        }
        if (!data_directory.isEmpty()) {
            for (const auto &ch : data_directory) {
                if (ch.unicode() < 0x20 || ch.unicode() == 0x7F) {
                    result.error = "Tor data_directory contains invalid control characters";
                    return result;
                }
            }
            if (data_directory.contains("..")) {
                result.error = "Tor data_directory must not contain '..'";
                return result;
            }
            outbound["data_directory"] = data_directory;
        }
        // C1 fix: Filter extra_args — reject args containing shell metacharacters
        if (!extra_args.isEmpty()) {
            const QString dangerous = ";|&`$(){}[]<>!\\\"";
            QStringList safeArgs;
            for (const auto &arg : extra_args) {
                bool hasDangerous = false;
                for (const auto &ch : arg) {
                    if (dangerous.contains(ch)) {
                        hasDangerous = true;
                        break;
                    }
                }
                if (!hasDangerous) {
                    safeArgs.append(arg);
                }
            }
            if (!safeArgs.isEmpty()) {
                outbound["extra_args"] = QList2QJsonArray(safeArgs);
            }
        }

        if (!torrc_json.isEmpty()) {
            auto torrcObj = QString2QJsonObject(torrc_json);
            if (!torrcObj.isEmpty()) {
                outbound["torrc"] = torrcObj;
            }
        }

        result.outbound = outbound;
        return result;
    }
} // namespace NekoGui_fmt
