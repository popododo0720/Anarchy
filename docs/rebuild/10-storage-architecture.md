# Anarchy Storage Architecture

Status: pre-development baseline

## Goal
Support pluggable storage backends without locking the product model to a single implementation.

## Required extensibility
Future storage backends may include:
- local storage
- NAS/NFS
- Ceph
- externally provided storage

Therefore storage must be port-driven.

## Core abstractions
- Volume
- VolumeAttachment
- ImageStore / ImageSource
- StorageBackend port
- VolumeProvisioner port

## Implementation strategy
Early implementation:
- one real local backend only

Design strategy:
- keep backend-specific behavior in adapters
- keep application/domain semantics backend-agnostic

## Rule
Do not hardcode local-storage assumptions into core business logic, even if local storage is the first implementation.
