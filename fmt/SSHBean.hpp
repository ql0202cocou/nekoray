#pragma once

#include "fmt/AbstractBean.hpp"

namespace NekoGui_fmt {
    class SSHBean : public AbstractBean {
    public:
        QString user = "";
        QString password = "";
        QStringList private_key;
        QString private_key_path = "";
        QString private_key_passphrase = "";
        QStringList host_key;
        QStringList host_key_algorithms;
        QString client_version = "";

        SSHBean() : AbstractBean(0) {
            _add(new configItem("user", &user, itemType::string));
            _add(new configItem("password", &password, itemType::string));
            _add(new configItem("private_key", &private_key, itemType::stringList));
            _add(new configItem("private_key_path", &private_key_path, itemType::string));
            _add(new configItem("private_key_passphrase", &private_key_passphrase, itemType::string));
            _add(new configItem("host_key", &host_key, itemType::stringList));
            _add(new configItem("host_key_algorithms", &host_key_algorithms, itemType::stringList));
            _add(new configItem("client_version", &client_version, itemType::string));
        };

        QString DisplayType() override { return "SSH"; };

        CoreObjOutboundBuildResult BuildCoreObjSingBox() override;

        bool TryParseLink(const QString &link);

        QString ToShareLink() override;
    };
} // namespace NekoGui_fmt
