with import <nixpkgs>{ config.allowUnfree = true; };
mkShell{
  name="survey-bot-test-shell";
  shellHook= ''
    source ${toString ./.env}
  '';

}

