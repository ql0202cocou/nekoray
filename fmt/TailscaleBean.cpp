#include "TailscaleBean.hpp"
#include <QJsonObject>
#include <QJsonArray>

namespace NekoGui_fmt {
    QJsonObject TailscaleBean::BuildEndpointConfig() {
        QJsonObject endpoint{
            {"type", "tailscale"},
        };

        if (!state_directory.isEmpty()) endpoint["state_directory"] = state_directory;
        if (!auth_key.isEmpty()) endpoint["auth_key"] = auth_key;
        if (!control_url.isEmpty()) endpoint["control_url"] = control_url;
        if (ephemeral) endpoint["ephemeral"] = true;
        if (!hostname.isEmpty()) endpoint["hostname"] = hostname;
        if (accept_routes) endpoint["accept_routes"] = true;
        if (!exit_node.isEmpty()) endpoint["exit_node"] = exit_node;
        if (exit_node_allow_lan_access) endpoint["exit_node_allow_lan_access"] = true;
        if (advertise_exit_node) endpoint["advertise_exit_node"] = true;

        if (!advertise_routes.isEmpty()) {
            QJsonArray routes;
            for (const auto &route : advertise_routes) {
                routes.append(route);
            }
            endpoint["advertise_routes"] = routes;
        }

        if (!advertise_tags.isEmpty()) {
            QJsonArray tags;
            for (const auto &tag : advertise_tags) {
                tags.append(tag);
            }
            endpoint["advertise_tags"] = tags;
        }

        return endpoint;
    }

    bool TailscaleBean::TryParseLink(const QString &link) {
        // Tailscale doesn't have a standard share link format
        Q_UNUSED(link);
        return false;
    }

    QString TailscaleBean::ToShareLink() {
        // Tailscale doesn't have a standard share link format
        return {};
    }
} // namespace NekoGui_fmt
