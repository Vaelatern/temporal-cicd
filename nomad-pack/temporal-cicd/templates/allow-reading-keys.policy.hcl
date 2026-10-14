#!/usr/bin/env -S nomad acl policy apply -namespace default -job temporal-cicd allow-reading-temporalcicd-keys

namespace "default" {
  variables {
    path "temporal-cicd/keys-d/*" {
       capabilities = ["read"]
     }
     path "temporal-cicd/shared-secrets/*" {
       capabilities = ["read"]
     }
   }
 }
