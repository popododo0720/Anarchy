# Anarchy Storage Backend Specification

Status: implementation draft

## 1. Goal
Ensure storage semantics remain stable while backend implementations vary.

## 2. Required backend compatibility direction
Potential backends:
- local storage
- NFS/NAS
- Ceph
- externally provided storage systems

## 3. Required ports
- StorageBackend
- VolumeProvisioner
- ImageSource
- ImageStore if separated later

## 4. Required rule
Domain and application logic must not assume local disk behavior.
Backend-specific concerns belong in adapters.

## 5. Initial implementation path
- implement LocalStorageAdapter first
- define tests and contracts so future adapters can be added without rewriting domain/application behavior
