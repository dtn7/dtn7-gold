"""
DTN7-go service for CORE Emulator.
"""
from typing import Tuple

from core.nodes.base import CoreNode
from core.services.coreservices import CoreService, ServiceMode


class DTNService(CoreService):
    """
    Example Custom CORE Service

    :cvar name: name used as a unique ID for this service and is required, no spaces
    :cvar group: allows you to group services within the GUI under a common name
    :cvar executables: executables this service depends on to function, if executable is
        not on the path, service will not be loaded
    :cvar dependencies: services that this service depends on for startup, tuple of
        service names
    :cvar dirs: directories that this service will create within a node
    :cvar configs: files that this service will generate, without a full path this file
        goes in the node's directory e.g. /tmp/pycore.12345/n1.conf/myfile
    :cvar startup: commands used to start this service, any non-zero exit code will
        cause a failure
    :cvar validate: commands used to validate that a service was started, any non-zero
        exit code will cause a failure
    :cvar validation_mode: validation mode, used to determine startup success.
        NON_BLOCKING    - runs startup commands, and validates success with validation commands
        BLOCKING        - runs startup commands, and validates success with the startup commands themselves
        TIMER           - runs startup commands, and validates success by waiting for "validation_timer" alone
    :cvar validation_timer: time in seconds for a service to wait for validation, before
        determining success in TIMER/NON_BLOCKING modes.
    :cvar validation_period: period in seconds to wait before retrying validation,
        only used in NON_BLOCKING mode
    :cvar shutdown: shutdown commands to stop this service
    """

    name: str = "dtnd"
    group: str = "dtn7"
    executables: Tuple[str, ...] = ("dtnd",)
    dependencies: Tuple[str, ...] = ()
    dirs: Tuple[str, ...] = ()
    configs: Tuple[str, ...] = ("dtnd.toml", )
    startup: Tuple[str, ...] = ("bash -c 'dtnd dtnd.toml &> dtnd.log'",)
    validate: Tuple[str, ...] = ("sleep 2",)
    validation_mode: ServiceMode = ServiceMode.NON_BLOCKING
    validation_timer: int = 5
    validation_period: float = 0.5
    shutdown: Tuple[str, ...] = ()

    @classmethod
    def generate_config(cls, node: CoreNode, filename: str) -> str:
        # "dtnd.toml"
        if filename == cls.configs[0]:
            return f"""
[core]
store = "store"
inspect-all-bundles = true
node-id = "dtn://{node.name}/"

[logging]
level = "info"
format = "text"

[discovery]
ipv4 = true
ipv6 = false

[agents]

[agents.webserver]
address = "localhost:8080"
websocket = true
rest = true

[[listen]]
protocol = "mtcp"
endpoint = ":4556"

[routing]
algorithm = "epidemic"
"""

class DTNExchangeService(CoreService):
    """
    Example Custom CORE Service

    :cvar name: name used as a unique ID for this service and is required, no spaces
    :cvar group: allows you to group services within the GUI under a common name
    :cvar executables: executables this service depends on to function, if executable is
        not on the path, service will not be loaded
    :cvar dependencies: services that this service depends on for startup, tuple of
        service names
    :cvar dirs: directories that this service will create within a node
    :cvar configs: files that this service will generate, without a full path this file
        goes in the node's directory e.g. /tmp/pycore.12345/n1.conf/myfile
    :cvar startup: commands used to start this service, any non-zero exit code will
        cause a failure
    :cvar validate: commands used to validate that a service was started, any non-zero
        exit code will cause a failure
    :cvar validation_mode: validation mode, used to determine startup success.
        NON_BLOCKING    - runs startup commands, and validates success with validation commands
        BLOCKING        - runs startup commands, and validates success with the startup commands themselves
        TIMER           - runs startup commands, and validates success by waiting for "validation_timer" alone
    :cvar validation_timer: time in seconds for a service to wait for validation, before
        determining success in TIMER/NON_BLOCKING modes.
    :cvar validation_period: period in seconds to wait before retrying validation,
        only used in NON_BLOCKING mode
    :cvar shutdown: shutdown commands to stop this service
    """

    name: str = "dtn-exchange"
    group: str = "dtn7"
    executables: Tuple[str, ...] = ("dtn-tool",)
    dependencies: Tuple[str, ...] = ("dtnd", )
    dirs: Tuple[str, ...] = ()
    configs: Tuple[str, ...] = ()
    startup: Tuple[str, ...] = ()
    validate: Tuple[str, ...] = ()
    validation_mode: ServiceMode = ServiceMode.NON_BLOCKING
    validation_timer: int = 5
    validation_period: float = 0.5
    shutdown: Tuple[str, ...] = ()

    @classmethod
    def get_startup(cls, node: CoreNode) -> Tuple[str, ...]:
        return (
            "mkdir dtn-exchange",
            f"bash -c 'dtn-tool exchange ws://localhost:8080/ws dtn://{node.name}/ dtn-exchange &> dtn-exchange.log'",
        )