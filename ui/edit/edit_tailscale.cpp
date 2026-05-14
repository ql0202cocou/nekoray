#include "edit_tailscale.h"
#include "ui_edit_tailscale.h"

#include "fmt/TailscaleBean.hpp"

EditTailscale::EditTailscale(QWidget *parent) : QWidget(parent), ui(new Ui::EditTailscale) {
    ui->setupUi(this);
}

EditTailscale::~EditTailscale() {
    delete ui;
}

void EditTailscale::onStart(std::shared_ptr<NekoGui::ProxyEntity> _ent) {
    this->ent = _ent;
    auto bean = this->ent->TailscaleBean();
    ui->state_directory->setText(bean->state_directory);
    ui->auth_key->setText(bean->auth_key);
    ui->control_url->setText(bean->control_url);
    ui->ephemeral->setChecked(bean->ephemeral);
    ui->hostname->setText(bean->hostname);
    ui->accept_routes->setChecked(bean->accept_routes);
    ui->exit_node->setText(bean->exit_node);
    ui->exit_node_allow_lan_access->setChecked(bean->exit_node_allow_lan_access);
    ui->advertise_routes->setText(bean->advertise_routes.join("\n"));
    ui->advertise_exit_node->setChecked(bean->advertise_exit_node);
    ui->advertise_tags->setText(bean->advertise_tags.join("\n"));
}

bool EditTailscale::onEnd() {
    auto bean = this->ent->TailscaleBean();
    bean->state_directory = ui->state_directory->text();
    bean->auth_key = ui->auth_key->text();
    bean->control_url = ui->control_url->text();
    bean->ephemeral = ui->ephemeral->isChecked();
    bean->hostname = ui->hostname->text();
    bean->accept_routes = ui->accept_routes->isChecked();
    bean->exit_node = ui->exit_node->text();
    bean->exit_node_allow_lan_access = ui->exit_node_allow_lan_access->isChecked();
    bean->advertise_routes = ui->advertise_routes->toPlainText().split("\n", Qt::SkipEmptyParts);
    bean->advertise_exit_node = ui->advertise_exit_node->isChecked();
    bean->advertise_tags = ui->advertise_tags->toPlainText().split("\n", Qt::SkipEmptyParts);
    return true;
}
