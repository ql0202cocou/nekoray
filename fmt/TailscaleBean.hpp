#pragma once

#include "fmt/AbstractBean.hpp"

namespace NekoGui_fmt {
    class TailscaleBean : public AbstractBean {
    public:
        QString state_directory = "";
        QString auth_key = "";
        QString control_url = "";
        bool ephemeral = false;
        QString hostname = "";
        bool accept_routes = false;
        QString exit_node = "";
        bool exit_node_allow_lan_access = false;
        QStringList advertise_routes;
        bool advertise_exit_node = false;
        QStringList advertise_tags;

        TailscaleBean() : AbstractBean(0) {
            _add(new configItem("state_directory", &state_directory, itemType::string));
            _add(new configItem("auth_key", &auth_key, itemType::string));
            _add(new configItem("control_url", &control_url, itemType::string));
            _add(new configItem("ephemeral", &ephemeral, itemType::boolean));
            _add(new configItem("hostname", &hostname, itemType::string));
            _add(new configItem("accept_routes", &accept_routes, itemType::boolean));
            _add(new configItem("exit_node", &exit_node, itemType::string));
            _add(new configItem("exit_node_allow_lan_access", &exit_node_allow_lan_access, itemType::boolean));
            _add(new configItem("advertise_routes", &advertise_routes, itemType::stringList));
            _add(new configItem("advertise_exit_node", &advertise_exit_node, itemType::boolean));
            _add(new configItem("advertise_tags", &advertise_tags, itemType::stringList));
        };

        QString DisplayType() override { return "Tailscale"; };

        QString DisplayAddress() override { return ""; };

        // Tailscale is an endpoint, not an outbound
        // BuildCoreObjSingBox returns endpoint config, not outbound config
        QJsonObject BuildEndpointConfig();

        bool TryParseLink(const QString &link);

        QString ToShareLink() override;
    };
} // namespace NekoGui_fmt
