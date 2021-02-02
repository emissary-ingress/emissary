import React from 'react';
import PropTypes from 'prop-types';
import { makeStyles } from '@material-ui/core/styles';
import AppBar from '@material-ui/core/AppBar';
import Tabs from '@material-ui/core/Tabs';
import Tab from '@material-ui/core/Tab';
import Typography from '@material-ui/core/Typography';
import Box from '@material-ui/core/Box';
import Button from '@material-ui/core/Button';
import HubspotForm from 'react-hubspot-form';

function TabPanel(props) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      {...other}
    >
      {value === index && (
        <Box p={3}>
          <Typography>{children}</Typography>
        </Box>
      )}
    </div>
  );
}

TabPanel.propTypes = {
  children: PropTypes.node,
  index: PropTypes.any.isRequired,
  value: PropTypes.any.isRequired,
};

function a11yProps(index) {
  return {
    id: `simple-tab-${index}`,
    'aria-controls': `simple-tabpanel-${index}`,
  };
}

const useStyles = makeStyles((theme) => ({
  root: {
    flexGrow: 1,
    backgroundColor: 'transparent',

  },

}));

export default function SimpleTabs() {
  const classes = useStyles();
  const [value, setValue] = React.useState(0);

  const handleChange = (event, newValue) => {
    setValue(newValue);
  };

  return (
    <div className={classes.root}>
      <AppBar elevation={0} style={{ background: 'transparent', color: 'black', borderBottom: '1px solid #e8e8e8', }} position="static">
        <Tabs TabIndicatorProps={{ style: { background: '#AF5CF8' } }} value={value} onChange={handleChange} aria-label="simple tabs example">
          <Tab label="macOS" {...a11yProps(0)} style={{ minWidth: "10%", textTransform: 'none' }} />
          <Tab label="Linux" {...a11yProps(1)} style={{ minWidth: "10%", textTransform: 'none' }} />
          <Tab label="Windows" {...a11yProps(2)} style={{ minWidth: "10%", textTransform: 'none' }} />
        </Tabs>
      </AppBar>
      <TabPanel value={value} index={0}>

        {/*macOS install instructions*/}
         To install Telepresence for macOS, run:
        <pre class="language-">
          <div class="token-line">
            <span class="token plain">bash sudo curl -fL https://s3.amazonaws.com/datawire-static-files/tel2/darwin/amd64/latest/telepresence /</span>
          </div>
          <div class="token-line">
            <span class="token plain">-o /usr/local/bin/telepresence /</span>
          </div>
          <div class="token-line">
            <span class="token plain">&& sudo chmod a+x /usr/local/bin/telepresence</span>
          </div>
        </pre>
        <Button style={{ textTransform: 'none' }} color="primary" variant="outlined" onClick={() => navigator.clipboard.writeText(
          'sudo curl -fL https://s3.amazonaws.com/datawire-static-files/tel2/darwin/amd64/latest/telepresence -o /usr/local/bin/telepresence && sudo chmod a+x /usr/local/bin/telepresence'
        )}>
          Copy Command
          </Button><br />

      Then check the version with this command to confirm the installation was successful.<br />
        <pre class="language-">
          <div class="token-line">
            <span class="token plain">telepresence version</span>
          </div></pre>
      If you receive an error that the app is from an unidentified developer, open <strong>System Preferences > Security & Privacy > General</strong>.  Click <strong>Open Anyway</strong> at the bottom to bypass the security block. Then retry the <code>telepresence version</code> command.

      </TabPanel>


      <TabPanel value={value} index={1}>

        {/*Linux install instructions*/}
        To install Telepresence for Linux, run:
        <pre class="language-">
          <div class="token-line">
            <span class="token plain">bash sudo curl -fL https://s3.amazonaws.com/datawire-static-files/tel2/linux/amd64/latest/telepresence /</span>
          </div>
          <div class="token-line">
            <span class="token plain">-o /usr/local/bin/telepresence /</span>
          </div>
          <div class="token-line">
            <span class="token plain">&& sudo chmod a+x /usr/local/bin/telepresence</span>
          </div>
        </pre>
        <Button style={{ textTransform: 'none' }} color="primary" variant="outlined" onClick={() => navigator.clipboard.writeText(
          'sudo curl -fL https://s3.amazonaws.com/datawire-static-files/tel2/linux/amd64/latest/telepresence -o /usr/local/bin/telepresence && sudo chmod a+x /usr/local/bin/telepresence'
        )}>
          Copy Command
          </Button><br />

      Then check the version with this command to confirm the installation was successful.<br />
        <pre class="language-">
          <div class="token-line">
            <span class="token plain">telepresence version</span>
          </div></pre>
      </TabPanel>

      <TabPanel value={value} index={2}>

        {/*Windows install instructions*/}
        Telepresence for Windows is coming soon, sign up here to notified when it is available.<br /><br />
        <HubspotForm
          portalId='485087'
          formId='2f542f1b-3da8-4319-8057-96fed78e4c26'
          onSubmit={() => console.log('Submit!')}
          onReady={(form) => console.log('Form ready!')}
          loading={<div>Loading...</div>}
        />

      </TabPanel>
    </div >
  );
}