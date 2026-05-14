#pragma once

#include <QWidget>
#include "profile_editor.h"

QT_BEGIN_NAMESPACE
namespace Ui {
    class EditTor;
}
QT_END_NAMESPACE

class EditTor : public QWidget, public ProfileEditor {
    Q_OBJECT

public:
    explicit EditTor(QWidget *parent = nullptr);

    ~EditTor() override;

    void onStart(std::shared_ptr<NekoGui::ProxyEntity> _ent) override;

    bool onEnd() override;

    QList<QPair<QPushButton *, QString>> get_editor_cached() override { return {{ui->torrc_edit, CACHE.torrc_json}}; };

private:
    Ui::EditTor *ui;
    std::shared_ptr<NekoGui::ProxyEntity> ent;

    struct {
        QString torrc_json;
    } CACHE;

private slots:
    void on_torrc_edit_clicked();
};
