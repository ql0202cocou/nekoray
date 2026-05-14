#pragma once

#include "main/NekoGui.hpp"
#include "db/traffic/TrafficData.hpp"
#include "fmt/AbstractBean.hpp"

namespace NekoGui_fmt {
    class SocksHttpBean;

    class ShadowSocksBean;

    class VMessBean;

    class TrojanVLESSBean;

    class NaiveBean;

    class QUICBean;

    class CustomBean;

    class ChainBean;

    class AnyTLSBean;

    class SSHBean;

    class TorBean;

    class TailscaleBean;
}; // namespace NekoGui_fmt

namespace NekoGui {
    class ProxyEntity : public JsonStore {
    public:
        QString type;

        int id = -1;
        int gid = 0;
        int latency = 0;
        std::shared_ptr<NekoGui_fmt::AbstractBean> bean;
        std::shared_ptr<NekoGui_traffic::TrafficData> traffic_data = std::make_shared<NekoGui_traffic::TrafficData>("");

        QString full_test_report;

        ProxyEntity(NekoGui_fmt::AbstractBean *bean, const QString &type_);

        [[nodiscard]] QString DisplayLatency() const;

        [[nodiscard]] QColor DisplayLatencyColor() const;

        [[nodiscard]] NekoGui_fmt::ChainBean *ChainBean() const {
            return static_cast<NekoGui_fmt::ChainBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::SocksHttpBean *SocksHTTPBean() const {
            return static_cast<NekoGui_fmt::SocksHttpBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::ShadowSocksBean *ShadowSocksBean() const {
            return static_cast<NekoGui_fmt::ShadowSocksBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::VMessBean *VMessBean() const {
            return static_cast<NekoGui_fmt::VMessBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::TrojanVLESSBean *TrojanVLESSBean() const {
            return static_cast<NekoGui_fmt::TrojanVLESSBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::NaiveBean *NaiveBean() const {
            return static_cast<NekoGui_fmt::NaiveBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::QUICBean *QUICBean() const {
            return static_cast<NekoGui_fmt::QUICBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::CustomBean *CustomBean() const {
            return static_cast<NekoGui_fmt::CustomBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::AnyTLSBean *AnyTLSBean() const {
            return static_cast<NekoGui_fmt::AnyTLSBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::SSHBean *SSHBean() const {
            return static_cast<NekoGui_fmt::SSHBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::TorBean *TorBean() const {
            return static_cast<NekoGui_fmt::TorBean *>(bean.get());
        };

        [[nodiscard]] NekoGui_fmt::TailscaleBean *TailscaleBean() const {
            return static_cast<NekoGui_fmt::TailscaleBean *>(bean.get());
        };
    };
} // namespace NekoGui
