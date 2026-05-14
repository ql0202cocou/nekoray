#pragma once

#include "fmt/AbstractBean.hpp"

namespace NekoGui_fmt {
    class TorBean : public AbstractBean {
    public:
        QString executable_path = "";
        QString data_directory = "";
        QStringList extra_args;
        QMap<QString, QString> torrc;
        QString torrc_json = "";

        TorBean() : AbstractBean(0) {
            _add(new configItem("executable_path", &executable_path, itemType::string));
            _add(new configItem("data_directory", &data_directory, itemType::string));
            _add(new configItem("extra_args", &extra_args, itemType::stringList));
            _add(new configItem("torrc", &torrc_json, itemType::string));
        };

        QString DisplayType() override { return "Tor"; };

        QString DisplayAddress() override { return ""; };

        CoreObjOutboundBuildResult BuildCoreObjSingBox() override;

        bool TryParseLink(const QString &link);

        QString ToShareLink() override;
    };
} // namespace NekoGui_fmt
