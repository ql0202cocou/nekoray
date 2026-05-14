#include "edit_tor.h"
#include "ui_edit_tor.h"

#include "fmt/TorBean.hpp"
#include "3rdparty/qv2ray/v2/ui/widgets/editors/w_JsonEditor.hpp"

EditTor::EditTor(QWidget *parent) : QWidget(parent), ui(new Ui::EditTor) {
    ui->setupUi(this);
}

QList<QPair<QPushButton *, QString>> EditTor::get_editor_cached() {
    return {{ui->torrc_edit, CACHE.torrc_json}};
}

EditTor::~EditTor() {
    delete ui;
}

void EditTor::onStart(std::shared_ptr<NekoGui::ProxyEntity> _ent) {
    this->ent = _ent;
    auto bean = this->ent->TorBean();
    ui->executable_path->setText(bean->executable_path);
    ui->data_directory->setText(bean->data_directory);
    ui->extra_args->setText(bean->extra_args.join("\n"));
    CACHE.torrc_json = bean->torrc_json;
}

bool EditTor::onEnd() {
    auto bean = this->ent->TorBean();
    bean->executable_path = ui->executable_path->text();
    bean->data_directory = ui->data_directory->text();
    bean->extra_args = ui->extra_args->toPlainText().split("\n", Qt::SkipEmptyParts);
    bean->torrc_json = CACHE.torrc_json;
    return true;
}

void EditTor::on_torrc_edit_clicked() {
    auto editor = new JsonEditor(QString2QJsonObject(CACHE.torrc_json), this);
    auto result = editor->OpenEditor();
    if (!result.isEmpty()) {
        CACHE.torrc_json = QJsonObject2QString(result, false);
    } else {
        CACHE.torrc_json = "";
    }
    editor_cache_updated();
}
