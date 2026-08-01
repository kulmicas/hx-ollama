from setuptools import setup, find_packages

setup(
    name="hx-ollama",
    version="0.1.0",
    description="Ollama AI integration tool for Helix Editor via Pipe (|) and Append/Insert Output",
    long_description=open("README.md", "r", encoding="utf-8").read() if os.path.exists("README.md") else "",
    long_description_content_type="text/markdown",
    author="Antigravity Team",
    url="https://github.com/mico/hx-ollama",
    packages=find_packages(),
    scripts=["bin/hx-ollama"],
    entry_points={
        "console_scripts": [
            "hx-ollama=hx_ollama.cli:main",
        ],
    },
    python_requires=">=3.8",
    classifiers=[
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: MIT License",
        "Operating System :: OS Independent",
    ],
)
