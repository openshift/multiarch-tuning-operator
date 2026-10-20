# The base image is expected to contain
# /bin/opm (with a serve subcommand) and /bin/grpc_health_probe
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
ARG FBC_VERSION
ARG OCP_VERSION

# Configure the entrypoint and command
ENTRYPOINT ["/bin/opm"]
CMD ["serve", "/configs", "--cache-dir=/tmp/cache"]

ADD licenses/ /licenses/
# Copy declarative config root into image at /configs and pre-populate serve cache
ADD ${FBC_VERSION}/catalog/ /configs
RUN ["/bin/opm", "serve", "/configs", "--cache-dir=/tmp/cache", "--cache-only"]

# Set DC-specific label for the location of the DC root directory
# in the image
LABEL operators.operatorframework.io.index.configs.v1=/configs
# TODO: when label is fully supported we can uncommnet; there is a known parsing bug
#LABEL com.redhat.fbc.openshift.version="[\"${OCP_VERSION}\"]"
