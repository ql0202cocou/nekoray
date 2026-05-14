#include "edit_ssh.h"
#include "ui_edit_ssh.h"

#include "fmt/SSHBean.hpp"

EditSSH::EditSSH(QWidget *parent) : QWidget(parent), ui(new Ui::EditSSH) {
    ui->setupUi(this);
}

EditSSH::~EditSSH() {
    delete ui;
}

void EditSSH::onStart(std::shared_ptr<NekoGui::ProxyEntity> _ent) {
    this->ent = _ent;
    auto bean = this->ent->SSHBean();
    ui->user->setText(bean->user);
    ui->password->setText(bean->password);
    ui->private_key_path->setText(bean->private_key_path);
    ui->private_key_passphrase->setText(bean->private_key_passphrase);
    ui->client_version->setText(bean->client_version);
    ui->private_key->setText(bean->private_key.join("\n"));
    ui->host_key->setText(bean->host_key.join("\n"));
    ui->host_key_algorithms->setText(bean->host_key_algorithms.join("\n"));
}

bool EditSSH::onEnd() {
    auto bean = this->ent->SSHBean();
    bean->user = ui->user->text();
    bean->password = ui->password->text();
    bean->private_key_path = ui->private_key_path->text();
    bean->private_key_passphrase = ui->private_key_passphrase->text();
    bean->client_version = ui->client_version->text();
    bean->private_key = ui->private_key->toPlainText().split("\n", Qt::SkipEmptyParts);
    bean->host_key = ui->host_key->toPlainText().split("\n", Qt::SkipEmptyParts);
    bean->host_key_algorithms = ui->host_key_algorithms->toPlainText().split("\n", Qt::SkipEmptyParts);
    return true;
}
