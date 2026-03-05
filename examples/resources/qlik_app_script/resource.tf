resource "qlik_app" "my_app" {
  name        = "My Application"
  description = "An application with a managed script"
}

resource "qlik_app_script" "my_script" {
  app_id           = qlik_app.my_app.id
  script           = file("${path.module}/scripts/main.qvs")
  version_message  = "Initial script version"
}

# Note: scripts/main.qvs is a file maintained in your repository alongside this configuration.
